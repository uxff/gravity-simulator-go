/*
	saver manager for calc_server
*/
package saver

import (
	"encoding/json"
	"log"
	"os"
	//"strconv"
	"strings"
	"time"

	"github.com/uxff/gravity-simulator-go/go_server/orbs"

	redigo "github.com/garyburd/redigo/redis"
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
	loadHandler SaverFace
	saveTimes   int
}

/*保存的接口声明*/
type SaverFace interface {
	SetConfig(config map[string]string) bool
	Save(key *string, val []byte) bool
	SaveList(key *string, oList []orbs.Orb) bool
	LoadList(key *string) []orbs.Orb
	Clear()
}
type FileSaver struct {
	savedir string
}
type McSaver struct {
	mc *memcache.Client
}
type RedisSaver struct {
	//client redis.Client
	client redigo.Conn
}

/*
	@param config["host"] = "mc://10.1.1.1:11211"
*/
func (this *McSaver) SetConfig(config map[string]string) bool {
	host, ok := config["host"]
	if ok {
		this.mc = memcache.New(host)
		return true
	} else {
		log.Println("empty config of mc saver")
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
func (this *McSaver) LoadList(cacheKey *string) (oList []orbs.Orb) {
	mc := this.mc

	if orbListStrVal, err := mc.Get(*cacheKey); err == nil {
		err := json.Unmarshal(orbListStrVal.Value, &oList)
		if err != nil {
			log.Println("mc.get len(val)=", len(orbListStrVal.Value), "after unmarshal, len=", len(oList), "json.Unmarshal err=", err)
		}
	} else {
		log.Println("mc.get", *cacheKey, "error:", err)
	}
	return oList
}
func (this *McSaver) Clear() {
	//this.mc.Close()
}

/*
	@param config["dir"] = "file://./filecache"
*/
func (this *FileSaver) SetConfig(config map[string]string) bool {
	dir, ok := config["dir"]
	if ok {
		this.savedir = dir
		return true
	} else {
		this.savedir = "./filecache/"
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
func (this *FileSaver) Clear() {
	// do none
}

/*
	@param config["host"] = "redis://10.1.1.1:6379"
*/
/* use alphazero/redis no connection pool
func (this *RedisSaver) SetConfig(config map[string]string) bool {
	spec := redis.DefaultSpec()
	hostAndPort := config["host"]
	hostAndPortArr := strings.Split(hostAndPort, ":")
	switch len(hostAndPortArr) {
	case 0:
		break
	case 1:
		spec.Host(hostAndPortArr[0])
	case 2:
		spec.Host(hostAndPortArr[0])
		port, _ := strconv.Atoi(hostAndPortArr[1])
		spec.Port(port)
	}
	spec.Db(0).Password("")

	client, e := redis.NewSynchClientWithSpec(spec)
	if e != nil {
		log.Println("connect to redis server failed:", e)
		return false
	}
	this.client = client
	return true
}*/
/*	use redigo, redis pool */
func (this *RedisSaver) SetConfig(config map[string]string) bool {

	hostAndPort := config["host"]
	if len(hostAndPort) == 0 {
		hostAndPort = "127.0.0.1:6379"
	}

	redisPool := &redigo.Pool{
		// 从配置文件获取maxidle以及maxactive，取不到则用后面的默认值
		MaxIdle:     5,
		MaxActive:   5,
		IdleTimeout: 180 * time.Second,
		Dial: func() (redigo.Conn, error) {
			c, err := redigo.Dial("tcp", hostAndPort)
			if err != nil {
				return nil, err
			}
			// 选择db
			c.Do("SELECT", 0)
			return c, nil
		},
	}
	this.client = redisPool.Get()

	return true
}
func (this *RedisSaver) SaveList(key *string, oList []orbs.Orb) bool {
	if strList, err := json.Marshal(oList); err == nil {
		return this.Save(key, strList)
	} else {
		log.Println("redis.set", *key, "json.Marshal error:", err)
	}
	return false
}
func (this *RedisSaver) Save(key *string, val []byte) bool {
	//err := this.client.Set(*key, val)
	_, err := this.client.Do("SET", *key, val)
	if err != nil {
		log.Println("redis.save failed:", err)
		return false
	}

	return true
}
func (this *RedisSaver) LoadList(cacheKey *string) (oList []orbs.Orb) {
	//orbListStr, err := this.client.Get(*cacheKey)
	orbListStr, err := redigo.Bytes(this.client.Do("GET", *cacheKey))

	if err == nil {
		err := json.Unmarshal(orbListStr, &oList)
		if err != nil {
			log.Println("redis.get len(val)=", len(orbListStr), "after unmarshal, len=", len(oList), "json.Unmarshal err=", err)
		}
	} else {
		log.Println("redis.get", *cacheKey, "error:", err)
	}
	return oList
}
func (this *RedisSaver) Clear() {
	this.client.Close()
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
	case 3:
		this.saveHandler = new(RedisSaver)
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
	if this.loadHandler != nil {
		return this.loadHandler.LoadList(key)
	} else {
		log.Println("loadpath not exist, so use savepath instead")
		return this.saveHandler.LoadList(key)
	}
}

// 将orbList存到数据库
func (this *Saver) SaveList(key *string, oList []orbs.Orb) bool {
	this.saveTimes++
	return this.saveHandler.SaveList(key, oList)
}
func (this *Saver) GetSavetimes() int {
	return this.saveTimes
}

/*
	@param savepath: like: "file://./filecahce","mc://10.0.0.1:11211","redis://10.1.1.1:6379"
*/
func (this *Saver) SetSavepath(savePath *string) {
	this.saveHandler = this.MakeSaverByPath(savePath)
	if this.loadHandler == nil {
		this.loadHandler = this.saveHandler
	}
}

/*
	@param schemePath: like: "file://./filecahce","mc://10.0.0.1:11211","redis://10.1.1.1:6379"
*/
func (this *Saver) SetLoadpath(schemePath *string) {
	this.loadHandler = this.MakeSaverByPath(schemePath)
}

/*
	根据schemePath获取SaverFace对应的对象
	其实属于静态方法
	@param schemePath: like: "file://./filecahce","mc://10.0.0.1:11211","redis://10.1.1.1:6379"
*/
func (this *Saver) MakeSaverByPath(schemePath *string) SaverFace {
	savePathCfg := strings.Split(*schemePath, "://")
	saverConf := make(map[string]string, 1)
	var saveHandler SaverFace
	if len(savePathCfg) > 1 {
		switch savePathCfg[0] {
		case "file":
			saverConf["dir"] = savePathCfg[1]
			saveHandler = new(FileSaver)
		case "mc":
			saverConf["host"] = savePathCfg[1]
			saveHandler = new(McSaver)
		case "redis":
			saverConf["host"] = savePathCfg[1]
			saveHandler = new(RedisSaver)
		default:
			log.Println("unknown save scheme:", savePathCfg[0])
		}
	} else {
		log.Println("illegal savePathCfg:", savePathCfg)
	}
	saveHandler.SetConfig(saverConf)
	return saveHandler
}

/*
	clear handler
*/
func (this *Saver) Clear() {
	this.saveHandler.Clear()
}
