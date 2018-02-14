package ipc

/*
#ifndef __IPC_H
#define __IPC_H
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <sys/ipc.h>
#include <sys/msg.h>
#include <sys/types.h>
#include <time.h>
#include <pthread.h>
#include <errno.h>
#include <stdbool.h>
#include <stdlib.h>

#define MSG_MAX 1024*3
#define KEY_SIZE 32

enum M_TYPE{SYSTEM = 1,DATA};

struct DATA_MSG{
    long mtype;
    long msgid;
    long msgcount;
    long seqid;
    char top_key[KEY_SIZE];
    char second_key[KEY_SIZE];
    char third_key[KEY_SIZE];
    char data[MSG_MAX];
};

struct SYS_MSG{
    long mtype;
    long msgid;
    long msgcount;
    long seqid;
    int systype;
    char data[MSG_MAX];
};

#endif

bool snd_data_msg(int msqid, struct DATA_MSG* msg){
    int sndlength = sizeof(struct DATA_MSG)-sizeof(long);
    int flag = msgsnd(msqid, msg, sndlength, 0);
    if(flag >= 0){
        return true;
    }
    printf("errno:%s",strerror(errno));
    return false;
}

bool snd_data(long msgid,long msgcount, long seqid, char* top_key, char* second_key, char* third_key, char* data){
    key_t key = ftok("/tmp/msg", 0x01);
    int msqid = msgget(key,IPC_CREAT|0600);
    struct DATA_MSG msg;

    msg.mtype = DATA;
    msg.msgid = msgid;
    msg.msgcount = msgcount;
    msg.seqid = seqid;
    strcpy(msg.top_key,top_key);
    strcpy(msg.second_key,second_key);
    strcpy(msg.third_key,third_key);
    strcpy(msg.data,data);
    return snd_data_msg(msqid,&msg);
}

*/
import "C"
import "unsafe"
import (
	"fmt"
)

func send_data_msg(msgid int64, msgcount int64, seqid int64, top_key string, second_key string, third_key string, data string) {
	top_key_f := C.CString(top_key)
	second_key_f := C.CString(second_key)
	third_key_f := C.CString(third_key)
	data_f := C.CString(data)
	result := C.snd_data(C.long(msgid), C.long(msgcount), C.long(seqid), top_key_f, second_key_f, third_key_f, data_f)
	C.free(unsafe.Pointer(top_key_f))
	C.free(unsafe.Pointer(second_key_f))
	C.free(unsafe.Pointer(third_key_f))
	C.free(unsafe.Pointer(data_f))
	fmt.Println(result)
}

/*func main() {
	send_data_msg(1, 1, 1, "1", "2", "3", "data")
}*/
