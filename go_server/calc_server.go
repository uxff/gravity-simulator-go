package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"time"

	orbs "./orbs"
	saverpkg "./saver"
)

// 默认最大天体数量
const MAX_PARTICLES = 100

// 默认计算步数
const FOR_TIMES = 10000

var saver = saverpkg.Saver{}

func main() {
	num_orbs := MAX_PARTICLES
	num_times := FOR_TIMES
	var numCpu int

	flag.IntVar(&num_orbs, "init-orbs", 0, "how many orbs init, do init when its value >1")
	flag.IntVar(&num_times, "calc-times", 100, "how many times calc")
	var eternal = flag.Float64("eternal", 15000.0, "the mass of eternal, 0 means no eternal")
	var doShowList = flag.Bool("showlist", false, "show orb list and exit")
	var configMass = flag.Float64("config-mass", 10.0, "the mass of orbs")
	var configWide = flag.Float64("config-wide", 1000.0, "the wide of orbs")
	var configVelo = flag.Float64("config-velo", 0.005, "the velo of orbs")
	var configStyle = flag.Int("config-style", 1, "the style of orbs distribute, 1=cube,2=disc,3=sphere")
	var configCpu = flag.Int("config-cpu", 0, "how many cpu u want use, 0=all")
	var savePath = flag.String("savepath", "mc://127.0.0.1:11211", "where to save, support mc/file/redis\n\tlike: file://./filecache/")
	var saveKey = flag.String("savekey", "thelist1", "key name for save, like key of memcache, or filename in save dir")

	// flags 读取参数，必须要调用 flag.Parse()
	flag.Parse()
	log.SetFlags(0)

	if *configCpu > 0 {
		numCpu = *configCpu
	} else {
		numCpu = runtime.NumCPU() - 1
	}
	runtime.GOMAXPROCS(numCpu)

	saver.SetSavepath(savePath)

	var oList []orbs.Orb

	// 根据时间设置随机数种子
	rand.Seed(int64(time.Now().Nanosecond()))

	if num_orbs > 0 {
		initConfig := orbs.InitConfig{*configMass, *configWide, *configVelo, *eternal, *configStyle}
		oList = orbs.InitOrbs(num_orbs, &initConfig)
	} else {
		oList = saver.GetList(saveKey)
		num_orbs = len(oList)
	}
	if *doShowList {
		fmt.Println(oList)
		return
	}

	fmt.Println("start calc, orbs:", num_orbs, "will times:", num_times*num_orbs*num_orbs, "use cpu core:", numCpu)

	realTimes, perTimes, tmpTimes := 0, 0, 0
	startTimeNano := time.Now().UnixNano()

	for i := 0; i < num_times; i++ {
		perTimes = orbs.UpdateOrbs(oList, i)
		realTimes += perTimes

		tmpTimes += perTimes
		if tmpTimes > 10000000 {
			saver.SaveList(saveKey, oList)
			if i%10 == 1 {
				oList = orbs.ClearOrbList(oList)
			}
			tmpTimes = 0
		}
	}

	oList = orbs.ClearOrbList(oList)
	//fmt.Println("when clear oList=", oList)

	endTimeNano := time.Now().UnixNano()
	timeUsed := float64(endTimeNano-startTimeNano) / 1000000000.0
	fmt.Println("after calc, orbs:", len(oList), "real times:", realTimes, "used time:", timeUsed, "sec", "CPS:", float64(realTimes)/timeUsed)
	orbs.ShowMonitorInfo()

	saver.SaveList(saveKey, oList)

	endTimeNano = time.Now().UnixNano()
	timeUsed2 := float64(endTimeNano-startTimeNano) / 1000000000.0
	fmt.Println("all used time with save:", timeUsed2, "sec, saveTimes=", saver.GetSavetimes(), "save per sec=", float64(saver.GetSavetimes())/timeUsed, "clearTimes=", orbs.GetClearTimes())
}
