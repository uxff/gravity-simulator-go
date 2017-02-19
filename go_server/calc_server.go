package main

import (
	//"bytes"
	//"encoding/json"
	"flag"
	"fmt"
	//"math"
	"math/rand"
	//"os"
	"runtime"
	//"strconv"
	"time"

	orbs "./orbs"
	saverpkg "./saver"
)

var saver = saverpkg.Saver{}

func main() {
	num_orbs := orbs.MAX_PARTICLES
	num_times := orbs.FOR_TIMES
	var eternal float64
	var mcHost, mcKey string
	var numCpu int

	flag.IntVar(&num_orbs, "init-orbs", 0, "how many orbs init, do init when its value >1")
	flag.IntVar(&num_times, "calc-times", 100, "how many times calc")
	flag.Float64Var(&eternal, "eternal", 15000.0, "the mass of eternal, 0 means no eternal")
	flag.StringVar(&mcHost, "mchost", "127.0.0.1:11211", "memcache server")
	flag.StringVar(&mcKey, "savekey", "thelist1", "key name save into memcache")
	var doShowList = flag.Bool("showlist", false, "show orb list and exit")
	var configMass = flag.Float64("config-mass", 10.0, "the mass of orbs")
	var configWide = flag.Float64("config-wide", 1000.0, "the wide of orbs")
	var configVelo = flag.Float64("config-velo", 0.005, "the velo of orbs")
	var configCpu = flag.Int("config-cpu", 0, "how many cpu u want use, 0=all")

	// flags 读取参数，必须要调用 flag.Parse()
	flag.Parse()

	if *configCpu > 0 {
		numCpu = *configCpu
	} else {
		numCpu = runtime.NumCPU() - 1
	}
	runtime.GOMAXPROCS(numCpu)

	var oList []orbs.Orb

	var htype int = 1
	saverConf := map[string]string{"dir": "./go_server/filecache"}
	//saverConf := map[string]string{"host": mcHost}
	saver.SetHandler(htype, saverConf)

	// 根据时间设置随机数种子
	rand.Seed(int64(time.Now().Nanosecond()))

	if num_orbs > 0 {
		initConfig := orbs.InitConfig{*configMass, *configWide, *configVelo, eternal}
		oList = orbs.InitOrbs(num_orbs, &initConfig)
	} else {
		oList = saver.GetList(&mcKey)
	}
	if *doShowList {
		fmt.Println(oList)
		return
	}
	num_orbs = len(oList)

	realTimes, perTimes, tmpTimes, saveTimes := 0, 0, 0, 0
	startTimeNano := time.Now().UnixNano()

	for i := 0; i < num_times; i++ {
		perTimes = orbs.UpdateOrbs(oList, i)
		realTimes += perTimes

		tmpTimes += perTimes
		if tmpTimes > 5000000 {
			saver.SaveList(&mcKey, oList)
			oList = orbs.ClearOrbList(oList)
			tmpTimes = 0
			saveTimes++
		}
	}

	oList = orbs.ClearOrbList(oList)
	//fmt.Println("when clear oList=", oList)

	endTimeNano := time.Now().UnixNano()
	timeUsed := float64(endTimeNano-startTimeNano) / 1000000000.0
	fmt.Println("core:", numCpu, " orbs:", num_orbs, len(oList), "times:", num_times, "real:", realTimes, "use time:", timeUsed, "sec", "CPS:", float64(realTimes)/timeUsed)
	orbs.ShowMonitorInfo()

	saver.SaveList(&mcKey, oList)
	saveTimes++

	endTimeNano = time.Now().UnixNano()
	timeUsed = float64(endTimeNano-startTimeNano) / 1000000000.0
	fmt.Println("all used time with save:", timeUsed, "sec, saveTimes=", saveTimes, "save per sec=", float64(saveTimes)/timeUsed)
}
