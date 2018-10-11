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
	"sync"

	"github.com/nwaples/rardecode"
)

func ReadLine(fileName string, handleLine func(string), handleEnd func()) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	if handleEnd != nil {
		defer handleEnd()
	}
	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		line = strings.TrimSpace(line)
		handleLine(line)
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
		log.Printf("will try %s", str)
		str = strings.TrimSpace(str)
		if str == "" {
			return
		}
		rangePwd <- str
	}, func() {
		log.Printf("read end")
		rangePwd <- ""
		close(rangePwd)
		//done <- true
	})

	wg := &sync.WaitGroup{}

	for {
		select {
		case pwd := <-rangePwd:
			if pwd == "" {
				log.Printf("end line, will over")
				goto end
			}

			//log.Printf("try password:%s", pwd)
			wg.Add(1)

			go func(pwd string) {
				defer wg.Done()

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
					//done <- true
					//goto end
					return
				}
				log.Printf("pass is not enumed, lineNoInRar=%d", lineNoInRar)
			}(pwd)

		case <-done:
			log.Printf("done occur")
			goto end
			//break
		}
	}

end:
	wg.Wait()
	// read rar end
}
