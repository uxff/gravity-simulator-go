/*
	saver for calc_server
*/
package saver

import (
	"encoding/json"
	"log"
	"os"

	orbs "../orbs"
	"github.com/bradfitz/gomemcache/memcache"
)

const (
	HTYPE_FILE  = 1
	HTYPE_MC    = 2
	HTYPE_REDIS = 3
)

type Saver struct {
	htype       int
	saveHandler SaverFace
	saveTimes   int
}

/*保存的接口声明*/
type SaverFace interface {
	SetConfig(config map[string]string) bool
	Save(key *string, val []byte) bool
	SaveList(key *string, oList []orbs.Orb) bool
	LoadList(key *string) []orbs.Orb
}
type FileSaver struct {
	savedir string
}
type McSaver struct {
	mc *memcache.Client
}

func (this *McSaver) SetConfig(config map[string]string) bool {
	host, ok := config["host"]
	if ok {
		this.mc = memcache.New(host)
		return true
	}
	return false
}
func (this *McSaver) SaveList(key *string, oList []orbs.Orb) bool {
	if strList, err := json.Marshal(oList); err == nil {
		return this.Save(key, strList)
	} else {
		log.Println("set", *key, "json.Marshal error:", err)
	}
	return false
}
func (this *McSaver) Save(key *string, val []byte) bool {
	errMc := this.mc.Set(&memcache.Item{Key: *key, Value: val})
	if errMc != nil {
		log.Println("save failed:", errMc)
		return false
	}

	return true
}
func (this *FileSaver) SetConfig(config map[string]string) bool {
	dir, ok := config["dir"]
	if ok {
		this.savedir = dir
		return true
	} else {
		this.savedir = "./go_server/filecache/"
	}
	return false
}

func (this *FileSaver) SaveList(key *string, oList []orbs.Orb) bool {
	if strList, err := json.Marshal(oList); err == nil {
		return this.Save(key, strList)
	} else {
		log.Println("set", *key, "json.Marshal error:", err)
	}
	return false
}

func (this *FileSaver) Save(key *string, val []byte) bool {
	var ret bool = false

	for {

		fileFullpath := this.savedir + "/" + *key
		cacheFile, errOpen := os.OpenFile(fileFullpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)

		if errOpen != nil {
			log.Println("open", fileFullpath, "error:", errOpen)
			break
		}
		defer cacheFile.Close()

		_, errW := cacheFile.Write(val)
		if errW != nil {
			log.Println("write file error:", errW)
			break
		}

		ret = true
		break
	}
	return ret
}

func (this *McSaver) LoadList(cacheKey *string) (oList []orbs.Orb) {
	var orbListStr string

	//var mc *memcache.Client = (*memcache.Client)(this.saveHandler)
	mc := this.mc
	if orbListStrVal, err := mc.Get(*cacheKey); err == nil {
		orbListStr = string(orbListStrVal.Value)
		err := json.Unmarshal(orbListStrVal.Value, &oList)
		log.Println("mc.get len(val)=", len(orbListStr), "after unmarshal, len=", len(oList), "json.Unmarshal err=", err)
	} else {
		log.Println("mc.get", *cacheKey, "error:", err)
	}
	return oList
}

func (this *FileSaver) LoadList(cacheKey *string) (oList []orbs.Orb) {
	var ret bool = false

	for {

		fileFullpath := this.savedir + "/" + *cacheKey
		cacheFile, errOpen := os.OpenFile(fileFullpath, os.O_RDONLY, os.ModePerm)

		if errOpen != nil {
			log.Println("open", fileFullpath, "error:", errOpen)
			break
		}
		defer cacheFile.Close()

		strContent := make([]byte, 1024)
		var allContent []byte
		for {
			n, _ := cacheFile.Read(strContent)
			if 0 == n {
				break
			}
			allContent = append(allContent, strContent[0:n]...)
		}
		errEnc := json.Unmarshal(allContent, &oList)
		if errEnc != nil {
			log.Println("json.Marshal error:", errEnc)
			break
		}
		ret = true
		break
	}
	if ret {
		return oList
	} else {
		return nil
	}
	//return oList
}
func (this *Saver) SetHandler(htype int, config map[string]string) (handler SaverFace) {
	this.htype = htype

	switch htype {
	case 1: //file
		this.saveHandler = new(FileSaver)
	case 2: //mc
		this.saveHandler = new(McSaver)
	default:
		log.Println("unknown htype:", htype)
	}
	this.saveHandler.SetConfig(config)
	return this.saveHandler

}
func (this *Saver) GetHandler() (handler SaverFace) {
	return this.saveHandler
}

// 从数据库获取orbList
func (this *Saver) GetList(key *string) (oList []orbs.Orb) {
	return this.saveHandler.LoadList(key)
}

// 将orbList存到数据库
func (this *Saver) SaveList(key *string, oList []orbs.Orb) bool {
	this.saveTimes++
	return this.saveHandler.SaveList(key, oList)
}
func (this *Saver) GetSavetimes() int {
	return this.saveTimes
}
