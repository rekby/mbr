package main
import (
	"os"
	"fmt"
	"github.com/rekby/mbr"
	"bytes"
)

func main(){
	dev := os.Args[1]
	f, err := os.Open(dev)
	defer f.Close()
	if err != nil {
		fmt.Println("File open error: " + err.Error())
	}

	Mbr, err := mbr.Read(f)
	if err != nil {
		fmt.Println("Error while read mbr: " + err.Error())
	}

	var buf *bytes.Buffer = &bytes.Buffer{}
	err = Mbr.Write(buf)
	if err != nil {
		fmt.Println("Error while write mbr to buf: " + err.Error())
	}

	fmt.Printf("MBR DUMP: %#v\n", buf.Bytes())
}