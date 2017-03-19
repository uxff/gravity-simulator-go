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
	var doShowList = flag.Bool("showlist", false, "show orb list and exit")
	var configMass = flag.Float64("config-mass", 10.0, "init mass of orbs")
	var configWide = flag.Float64("config-wide", 1000.0, "init wide of orbs")
	var configVelo = flag.Float64("config-velo", 0.005, "init velo of orbs")
	var configStyleArrange = flag.Int("config-arrange", 3, "init style of orbs arrangement 0=line,1=cube,2=disc,3=sphere")
	var configStyleAssemble = flag.Int("config-assemble", 2, "init style of orbs aggregation 0=avg,1=ladder,2=variance,3=4th power")
	var bigMass = flag.Float64("bigmass", 15000.0, "config of big mass orb, like a blackhole in center, 0 means no bigger")
	var bigNum = flag.Int("bignum", 1, "config of number of big mass orbs, generally center has 1")
	var bigMassStyle = flag.Int("bigstyle", 0, "config of big mass orb distribute style: 0=center 1=outer edge 2=middle of one radius 3=random")
	var configCpu = flag.Int("config-cpu", 0, "how many cpu u want use, 0=all")
	var savePath = flag.String("savepath", "mc://127.0.0.1:11211", "where to save, support mc/file/redis\n\tlike: file://./filecache/")
	var saveKey = flag.String("savekey", "thelist1", "key name to save, like key of memcache, or filename in save dir")
	var loadKey = flag.String("loadkey", "thelist1", "key name to load, like key of memcache, or filename in save dir")

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
		initConfig := orbs.InitConfig{*configMass, *configWide, *configVelo, *configStyleArrange, *configStyleAssemble, *bigMass, *bigNum, *bigMassStyle}
		oList = orbs.InitOrbs(num_orbs, &initConfig)
	} else {
		oList = saver.GetList(loadKey)
		num_orbs = len(oList)
	}
	if *doShowList {
		fmt.Println(oList)
		return
	}

	fmt.Printf("start calc, orbs:%d will times:%d use cpu core:%d allMass=%e\n", num_orbs, num_times*num_orbs*num_orbs, numCpu, orbs.GetAllMass())

	realTimes, perTimes, tmpTimes := 0, 0, 0
	startTimeNano := time.Now().UnixNano()

	for i := 0; i < num_times; i++ {
		perTimes = orbs.UpdateOrbs(oList, i)
		realTimes += perTimes

		tmpTimes += perTimes
		if tmpTimes > 10000000 {
			saver.SaveList(saveKey, oList)
			if i%10 == 1 { //orbs.GetCrashed()%10 == 9 &&
				oList = orbs.ClearOrbList(oList)
			}
			tmpTimes = 0
		}
	}

	oList = orbs.ClearOrbList(oList)
	//fmt.Println("when clear oList=", oList)

	endTimeNano := time.Now().UnixNano()
	timeUsed := float64(endTimeNano-startTimeNano) / 1000000000.0
	fmt.Printf("after calc, orbs:%d real times:%d used time:%6fs CPS:%e\n", len(oList), realTimes, timeUsed, float64(realTimes)/timeUsed)
	orbs.ShowMonitorInfo()

	saver.SaveList(saveKey, oList)

	endTimeNano = time.Now().UnixNano()
	timeUsed2 := float64(endTimeNano-startTimeNano) / 1000000000.0
	fmt.Printf("all used time with save:%6fs saveTimes:%d save/sec:%.2f clearTimes:%d crashed:%d\n", timeUsed2, saver.GetSavetimes(), float64(saver.GetSavetimes())/timeUsed, orbs.GetClearTimes(), orbs.GetCrashed())

	saver.Clear()
}

/*
	todo list:
	实现多服务器计算
*/
