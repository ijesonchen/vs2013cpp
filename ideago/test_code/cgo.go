package main

/*
1. import "C" 必须在嵌入的C代码后面的第一行，中间不能有空格，否则会提示could not determine kind of name
2. 通过C.CString(s)生成的char* p会产生一次内存拷贝，修改p不会影响s
3. printf不含有\n时，不会flush到输出，可能会导致cgo调用中printf不显示。如果\n后面有内容也可能不显示。可以使用fflush(stdout)强制刷新。
4. go unsafe.Pointer -> c void*
5. b []byte -> cgo void*
   unsafe.Pointer p = C.CBytes(b) -> cgo void*
   会在C堆上分配内存。使用完成后需要调用C.free(p)
6. cgo的指针传递
   https://studygolang.com/articles/3113 Go和C如何共享内存资源
   https://segmentfault.com/a/1190000013590585 cgo的指针传递
   简单来说，go分配的内存，不能通过指针传递给cgo。包括go调用cgo，或者cgo调用go时返回。
   如果通过unsafe包直接获取到内存地址，可以绕过限制，但是可能有风险。？gc会导致slice的移动吗？
   例如b []byte
     unsafe.Pointer(&b[0])可以传递给cgo void*，因为拿到了raw buf
     unsafe.Pointer(&b)不可以可以传递，因为是go memory
     也可以转换为SliceHeader,取Data（uintptr），然后转换为unsafe.Pointer:
     p := (*reflect.SliceHeader)(unsafe.Pointer(&b)), unsafe.Pointer(p.Data) -> cgo void*
   1) Go调用C Code时，Go传递给C Code的Go指针所指的Go Memory中不能包含任何指向Go Memory的Pointer。否则会出现panic
	panic: runtime error: cgo argument has Go pointer to Go pointer
7. 从测试来看，go中的float32和c中的float数值表示方式一致。
8. 可以使用runtime.SetFinalizer来管理资源（还没仔细研究）
   关键词 Go语言资源自动回收技术
   https://zhuanlan.zhihu.com/p/76504936 使用runtime.SetFinalizer优雅关闭后台goroutine
   https://mlog.club/article/90611  垃圾收集和cgo
   https://juejin.im/post/5d74783bf265da03ca11923d 深入理解Go-runtime.SetFinalizer原理剖析

*/

/*
#include <stdio.h>
#include <stdlib.h>
typedef long long int int64;

void Print(char* s){
	printf("input : %s\n", s);
	s[0]='*';
	printf("modify: %s\n", s);
	fflush(stdout);
}

void PrintV2(void* p){
	char* s = (char*)p;
	printf("input : %s\n", s);
	s[0]='*';
	printf("modify: %s\n", s);
	fflush(stdout);
}
void PrintSliceData(void* p){
	char* s;
	s = (char*)p;
	printf("input : %s\n", s);
	s[0]='*';
	printf("modify: %s\n", s);
	fflush(stdout);
}

void PrintSlice(void* p){
	char* s;
	s = (char*)p;
	printf("input : %s\n", s);
	s[0]='*';
	printf("modify: %s\n", s);
	fflush(stdout);
}

void printFloat(float* p, size_t nLen){
	for (size_t i=0; i<nLen;i++){
		printf("%.12f ", p[i]);
	}
	puts("");
}

void ProcFloat(void* pv, long long nLen){
	float* p;
	p = (float*)pv;
	printf("input : ");
	printFloat(p, nLen);
	float d = 1;
	for (size_t i=0; i<nLen;i++){
		p[i]= d/3;
		d = p[i];
	}
	printf("modify: ");
	printFloat(p, nLen);
	fflush(stdout);
}

char* NewData(){
	char* p = malloc(1024*1024);
	p[0]='a';
	p[1]=0;
	printf("data %s\n",p);
	//free(p);
	fflush(stdout);
	return p;
}

void FreeData(char* p){
	printf("data %s\n", p);
	printf("free ptr %p\n", p);
	free(p);
}


int LoadModel(const char* fnHash, const char* fnEmbed){
	printf("LoadModel %s %s\n", fnHash, fnEmbed);
	return 0;
}

#define uint64_t unsigned long long

int Feature2Embedding(char* workers, char* feaConf, uint64_t feaConfByte,
        char* ucFea, uint64_t ucFeaByte,
        char** ppItemFeas, uint64_t* pItemFeaBytes, uint64_t itemCount,
        float** ppEmbeds, uint64_t embedSize){
	printf("u %zd ul %zd ull %zd\n", sizeof(unsigned), sizeof(unsigned long), sizeof(unsigned long long));

	printf("Feature2Embedding =========>\n workers %s\n", workers);
	feaConf[feaConfByte-1]=0;
	printf("feaconf %s\n", feaConf);
	for(size_t i=0;i<itemCount;i++){
		ppItemFeas[i][pItemFeaBytes[i]-1]=0;
		printf("item %zd %s\n", i, ppItemFeas[i]);
		for (size_t j = 0;j<embedSize;j++){
			//printf("==> %p %p\n", ppEmbeds, ppEmbeds[i]);
			ppEmbeds[i][j] = 1.0/(i+j+1);
		}
	}
return 0;
}
*/
import "C"

import (
	"bytes"
	"fmt"
	"log"
	"reflect"
	"unsafe"
)

func getCPtr(ps *string) (p *C.char, n C.ulonglong) {
	pps := (*reflect.StringHeader)(unsafe.Pointer(ps))
	p = (*C.char)(unsafe.Pointer(pps.Data))
	n = (C.ulonglong)(pps.Len)
	return
}

func getCAPtr(ps *[]string, pa []uintptr, pan []C.ulonglong) (pp **C.char, pn *C.ulonglong) {
	for i := range *ps {
		p, n := getCPtr(&(*ps)[i])
		pa[i] = uintptr(unsafe.Pointer(p))
		pan[i] = n
	}
	pp = (**C.char)(unsafe.Pointer((*reflect.SliceHeader)(unsafe.Pointer(&pa)).Data))
	pn = (*C.ulonglong)(unsafe.Pointer((*reflect.SliceHeader)(unsafe.Pointer(&pan)).Data))
	return
}
func getFAPtr(ps *[][]float32, pa []uintptr) (pp **C.float) {
	for i := range *ps {
		pa[i] = (*reflect.SliceHeader)(unsafe.Pointer(&(*ps)[i])).Data
	}
	log.Println("pa:", pa)
	pp = (**C.float)(unsafe.Pointer((*reflect.SliceHeader)(unsafe.Pointer(&pa)).Data))
	return
}

func testFeature2Embedding() {
	var buf bytes.Buffer

	buf.WriteString("workdata")
	buf.WriteByte(0)
	worker := buf.String()

	buf.Reset()
	buf.WriteString("feaconf")
	feaConf := buf.String()

	buf.Reset()
	buf.WriteString("ucfea")
	ucFea := buf.String()

	var nItem C.ulonglong = 3
	iFeas := make([]string, nItem)
	for i := 0; i < int(nItem); i++ {
		buf.Reset()
		buf.WriteString(fmt.Sprintf("itemFea_%d_", i))
		for j := 0; j < i; j++ {
			buf.WriteString("x")
		}
		iFeas[i] = buf.String()
	}

	var embedSize C.ulonglong = 5
	embed := make([][]float32, nItem)
	for i := 0; i < int(nItem); i++ {
		embed[i] = make([]float32, embedSize)
	}

	p1, _ := getCPtr(&worker)
	p2, s2 := getCPtr(&feaConf)
	p3, s3 := getCPtr(&ucFea)
	ppfa := make([]uintptr, len(iFeas))
	ppfan := make([]C.ulonglong, len(iFeas))
	ppf1, pnf1 := getCAPtr(&iFeas, ppfa, ppfan)
	ppea := make([]uintptr, len(embed))
	ppe1 := getFAPtr(&embed, ppea)

	C.Feature2Embedding(p1, p2, s2, p3, s3, ppf1, pnf1, nItem, ppe1, embedSize)
	log.Printf("%T %T %T", ppfa, ppfan, ppea)
	log.Println(embed)
}

func testCgo() {
	testFeature2Embedding()
	return
	var buf bytes.Buffer
	buf.WriteString("this is a test")
	s := buf.String()
	log.Printf("s01 %s", s)
	C.Print(C.CString(s))
	log.Printf("s02 %s", s)

	log.Printf("ss01 %s", s)
	ps := (*reflect.StringHeader)(unsafe.Pointer(&s))
	C.PrintV2(unsafe.Pointer(ps.Data))
	log.Printf("ss02 %s", s)
	log.Println()

	b := buf.Bytes()
	b = append(b, 0)
	log.Printf("b101 %s", b)
	pb := C.CBytes(b)
	C.PrintSliceData(pb)
	C.free(pb)
	log.Printf("b102 %s", b)
	/*
		b2 := buf.Bytes()
		b2 = append(b2, 0)
		pb2 := unsafe.Pointer(&b2[0])
		log.Printf("b201 %d %s", len(b2), b2)
		C.PrintSlice(pb2)
		log.Printf("b202 %d %s", len(b2), b2)
	*/
	b3 := buf.Bytes()
	b3 = append(b3, 0)
	pb3 := (*reflect.SliceHeader)(unsafe.Pointer(&b3))
	log.Printf("b301 %d %s", len(b3), b3)
	C.PrintSlice(unsafe.Pointer(pb3.Data))
	log.Printf("b302 %d %s", len(b3), b3)

	fData := make([]float32, 10)
	pf := (*reflect.SliceHeader)(unsafe.Pointer(&fData))
	C.ProcFloat(unsafe.Pointer(pf.Data), C.longlong(pf.Len))
	log.Printf("float data %v", fData)

}
