package singularity

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	. "github.com/JasonYangShadow/lpmx/error"
	. "github.com/JasonYangShadow/lpmx/paeudo"
	. "github.com/JasonYangShadow/lpmx/utils"
	sif "github.com/sylabs/sif/pkg/sif"
)

type SingularityObject struct {
	ID       uint32
	Groupid  uint32
	Link     uint32
	Fileoff  int64
	Filelen  int64
	Fstype   string
	Partype  string
	Archtype string
}

func fstypeStr(ftype sif.Fstype) string {
	switch ftype {
	case sif.FsSquash:
		return "Squashfs"
	case sif.FsExt3:
		return "Ext3"
	case sif.FsImmuObj:
		return "Archive"
	case sif.FsRaw:
		return "Raw"
	case sif.FsEncryptedSquashfs:
		return "Encrypted squashfs"
	}
	return "Unknown fs-type"
}

func parttypeStr(ptype sif.Parttype) string {
	switch ptype {
	case sif.PartSystem:
		return "System"
	case sif.PartPrimSys:
		return "*System"
	case sif.PartData:
		return "Data"
	case sif.PartOverlay:
		return "Overlay"
	}
	return "Unknown part-type"
}

func trimZeroBytes(str []byte) string {
	return string(bytes.TrimRight(str, "\x00"))
}

func loadSif(path string) (*sif.FileImage, *Error) {
	fileimage, err := sif.LoadContainer(path, true)
	if err != nil {
		cerr := ErrNew(err, fmt.Sprintf("could not load image: %s", path))
		return nil, cerr
	}
	return &fileimage, nil
}

func unloadSif(fileimage *sif.FileImage) *Error {
	fileimage.UnloadContainer()
	return nil
}

func getDescriptorInfo(fileimage *sif.FileImage) ([]*SingularityObject, *Error) {
	sigs := []*SingularityObject{}
	for _, v := range fileimage.DescrArr {
		if v.Used && v.Datatype == sif.DataPartition {
			sig := new(SingularityObject)
			sig.ID = v.ID
			sig.Groupid = v.Groupid &^ sif.DescrGroupMask
			sig.Fileoff = v.Fileoff
			sig.Filelen = v.Filelen
			if v.Link != sif.DescrUnusedLink {
				if v.Link&sif.DescrGroupMask == sif.DescrGroupMask {
					sig.Link = v.Link &^ sif.DescrGroupMask
				} else {
					sig.Link = v.Link
				}
			}
			fstype, _ := v.GetFsType()
			partype, _ := v.GetPartType()
			archtype, _ := v.GetArch()
			sig.Fstype = fstypeStr(fstype)
			sig.Partype = parttypeStr(partype)
			sig.Archtype = sif.GetSIFArch(trimZeroBytes(archtype[:]))

			sigs = append(sigs, sig)
		}
	}
	return sigs, nil
}

func LoadSquashfs(squashfsfile, imagedir string) (map[string]int64, []string, *Error) {
	if !FileExist(squashfsfile) {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("could not find file: %s", squashfsfile))
		return nil, nil, cerr
	}
	sha256, err := Sha256file(squashfsfile)
	if err != nil {
		return nil, nil, err
	}

	layer_data := make(map[string]int64)
	var layers []string
	target_path := fmt.Sprintf("%s/%s", imagedir, sha256)
	err = Rename(squashfsfile, target_path)
	if err != nil {
		return nil, nil, err
	}
	file_length, ferr := GetFileLength(target_path)
	if ferr != nil {
		return nil, nil, ferr
	}
	layer_data[target_path] = file_length
	layers = append(layers, target_path)

	return layer_data, layers, nil
}

func ExtractSquashfs(sif, filepath string) *Error {
	if FileExist(filepath) {
		return nil
	}

	to, err := os.Create(filepath)
	defer to.Close()

	if err != nil {
		cerr := ErrNew(err, fmt.Sprintf("could not create file: %s", filepath))
		return cerr
	}

	fileimage, ferr := loadSif(sif)
	defer unloadSif(fileimage)
	if ferr != nil {
		return ferr
	}
	sings, serr := getDescriptorInfo(fileimage)
	if serr != nil {
		return serr
	}

	if sings != nil {
		fileoff := sings[0].Fileoff
		filelen := sings[0].Filelen

		if _, err := fileimage.Fp.Seek(fileoff, 0); err != nil {
			cerr := ErrNew(err, fmt.Sprintf("could not set offset of file: %s", filepath))
			return cerr
		}

		if _, err := io.CopyN(to, fileimage.Fp, filelen); err != nil {
			cerr := ErrNew(err, fmt.Sprintf("could not write data to file: %s", filepath))
			return cerr
		}
	}

	return nil
}

func Unsquashfs(squashfspath, destfolder string) *Error {
	if !FileExist(squashfspath) {
		err := ErrNew(ErrNExist, fmt.Sprintf("%s does not exist", squashfspath))
		return err
	}

	if !FolderExist(destfolder) {
		_, err := MakeDir(destfolder)
		if err != nil {
			return err
		}
	}

	_, err := Command("unsquashfs", "-f", "-d", destfolder, squashfspath)
	if err != nil {
		return err
	}

	//scan the destfolder and remove singularity related symlinks
	ferr := filepath.Walk(destfolder, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if file == destfolder {
			return nil
		}

		mode := fi.Mode()
		if mode&os.ModeSymlink != 0 {
			link, err := os.Readlink(file)
			if err != nil {
				return err
			}
			if strings.HasPrefix(link, ".singularity.d") {
				_, rerr := RemoveFile(file)
				if rerr != nil {
					return rerr.Err
				}
			}
		}

		return nil
	})

	singularity_folder := fmt.Sprintf("%s/.singularity.d", destfolder)
	if FolderExist(singularity_folder) {
		_, err := RemoveAll(singularity_folder)
		if err != nil {
			return err
		}
	}

	if ferr != nil {
		return ErrNew(ferr, fmt.Sprintf("could not unsquash file %s to folder: %s", squashfspath, destfolder))
	}
	return nil
}
