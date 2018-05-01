/*
Usage:
	-r="a.rar": rar file
	-w="weakpwd.txt": weak password txt
*/
package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"os"
	"strings"

	"github.com/nwaples/rardecode"
)

func ReadLine(fileName string, handler func(string)) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		line = strings.TrimSpace(line)
		handler(line)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
	return nil
}

func main() {
	rarFileName := flag.String("r", "a.rar", "rar file name")
	//password := flag.String("p", "", "password of rar file")
	weaklib := flag.String("w", "", "weak lib file of password")
	flag.Parse()

	// read rar start

	rangePwd := make(chan string, 1000)
	done := make(chan bool, 1)

	go ReadLine(*weaklib, func(str string) {
		rangePwd <- str
	})

	for {
		select {
		case pwd := <-rangePwd:
			if pwd == "" {
				break
			}

			//log.Printf("try password:%s", pwd)

			go func(pwd string) {

				rarreader, err := rardecode.OpenReader(*rarFileName, pwd)
				if err != nil {
					log.Printf("rar decode error:%v", err)
					return
				}
				defer rarreader.Close()

				isOk := false
				lineNoInRar := 0
				for {
					fhandler, err := rarreader.Next()
					if err == io.EOF {
						//log.Printf("normal end , break")
						isOk = true
						break
					}

					if err != nil {
						log.Printf("rar read next error:%v use pwd=%s fhandler=%s", err, pwd, fhandler)
						break
					}

					//fhandler.Mode()
					//log.Printf("read a file from fhandler:%v", fhandler.Name)
					lineNoInRar++
				}

				if isOk == true {
					log.Printf("the legal pass=%s", pwd)
					done <- true
				}
			}(pwd)

		case <-done:
			break
		}
	}

	// read rar end
}
