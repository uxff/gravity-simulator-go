/*
	天体及天体计算框架
*/
package orbs

import (
	"log"
	"math"
	"math/rand"
)

// 天体结构体声明
type Orb struct {
	X    float64 `json:"x"`  // 坐标x
	Y    float64 `json:"y"`  // 坐标y
	Z    float64 `json:"z"`  // 坐标z
	Vx   float64 `json:"vx"` // 速度x
	Vy   float64 `json:"vy"` // 速度y
	Vz   float64 `json:"vz"` // 速度z
	Mass float64 `json:"m"`  // 质量
	Size int     `json:"sz"` // 大小，用于计算吞并的天体数量
	Stat int     `json:"st"` // 用于标记是否已爆炸 1=正常 2=已爆炸
	Id   int     `json:"id"`
	//CalcTimes int     `json:"calcTimes"`
}

// 加速度结构体
type Acc struct {
	Ax float64
	Ay float64
	Az float64
	A  float64
}

// 配置
type InitConfig struct {
	Mass    float64
	Wide    float64
	Velo    float64
	Eternal float64
}

// 万有引力常数
const G = 0.000021

// 默认最大天体数量
const MAX_PARTICLES = 100

// 默认计算步数
const FOR_TIMES = 10000

// 最小天体距离值 两天体距离小于此值了会相撞
const MIN_CRITICAL_DIST = 1.0

// 监控速度和加速度
var maxVeloX, maxVeloY, maxVeloZ, maxAccX, maxAccY, maxAccZ, maxMass float64 = 0, 0, 0, 0, 0, 0, 0
var maxMassId int = 0

// 初始化天体位置，质量，加速度 在一片区域随机分布
func InitOrbs(num int, config *InitConfig) []Orb {
	oList := make([]Orb, num)

	for i := 0; i < num; i++ {
		o := &oList[i]

		o.X, o.Y, o.Z = (0.5-rand.Float64())*config.Wide, (0.5-rand.Float64())*config.Wide, (0.5-rand.Float64())*config.Wide
		o.Vx = (rand.Float64() - 0.5) * config.Velo * 2.0
		o.Vy = (rand.Float64() - 0.5) * config.Velo * 2.0
		o.Vz = (rand.Float64() - 0.5) * config.Velo * 2.0
		o.Size = 1
		o.Mass = rand.Float64() * config.Mass
		o.Id = i // rand.Int()
		o.Stat = 1
	}
	// 如果配置了恒星，将最后一个设置为恒星
	if config.Eternal != 0.0 {
		eternalId := num - 1
		eternalOrb := &oList[eternalId]
		eternalOrb.Mass = config.Eternal
		eternalOrb.Id = eternalId //rand.Int()
		eternalOrb.X, eternalOrb.Y, eternalOrb.Z = 0, 0, 0
	}
	return oList
}

// 所有天体运动一次
func UpdateOrbs(oList []Orb, nStep int) int {
	thelen := len(oList)
	c := make(chan int)
	cCount := 0
	for i := 0; i < thelen; i++ {
		go oList[i].Update(oList, c, nStep)
	}
	for {
		if cCount >= thelen {
			break
		}
		cCount += <-c
		//cCount += 1
	}
	// 以下方法运行时报错
	//	for cval := range c {
	//		cCount += cval
	//	}
	return cCount * cCount
}

// 天体运动一次
func (o *Orb) Update(oList []Orb, c chan int, nStep int) {
	aAll := o.CalcGravityAll(oList)
	if o.Stat == 1 {
		o.Vx += aAll.Ax
		o.Vy += aAll.Ay
		o.Vz += aAll.Az
		o.X += o.Vx
		o.Y += o.Vy
		o.Z += o.Vz
		// 监控速度和加速度
		if maxVeloX < math.Abs(o.Vx) {
			maxVeloX = math.Abs(o.Vx)
		}
		if maxVeloY < math.Abs(o.Vy) {
			maxVeloY = math.Abs(o.Vy)
		}
		if maxVeloZ < math.Abs(o.Vz) {
			maxVeloZ = math.Abs(o.Vz)
		}
		if maxAccX < math.Abs(aAll.Ax) {
			maxAccX = math.Abs(aAll.Ax)
		}
		if maxAccY < math.Abs(aAll.Ay) {
			maxAccY = math.Abs(aAll.Ay)
		}
		if maxAccZ < math.Abs(aAll.Az) {
			maxAccZ = math.Abs(aAll.Az)
		}
		if maxMass < o.Mass {
			maxMass = o.Mass
			maxMassId = o.Id
		}
	}
	//o.CalcTimes += 1
	c <- 1 //len(oList)
}

// 计算天体受到的总体引力
func (o *Orb) CalcGravityAll(oList []Orb) Acc {
	var gAll Acc
	for i := 0; i < len(oList); i++ {
		//c <- 1
		target := &oList[i]
		if target.Id == o.Id || target.Stat != 1 || o.Stat != 1 || o.Mass == 0 || target.Mass == 0 {
			continue
		}

		dist := o.CalcDist(target)

		// 距离太近，被撞
		isTooNearly := dist*dist < MIN_CRITICAL_DIST*MIN_CRITICAL_DIST
		// 速度太快，被撕裂
		isTaRipped := dist*dist < (target.Vx*target.Vx+target.Vy*target.Vy+target.Vz*target.Vz)*10

		if isTooNearly || isTaRipped {

			// 碰撞机制 非弹性碰撞 动量守恒 m1v1+m2v2=(m1+m2)v
			if o.Mass > target.Mass {
				//log.Println(o.Id, "crashed", target.Id, "isTooNearly", isTooNearly, "me=", o, "ta=", target)
				// 碰撞后速度 v = (m1v1+m2v2)/(m1+m2)
				o.Mass += target.Mass
				o.Vx = (target.Mass*target.Vx + o.Mass*o.Vx) / o.Mass
				o.Vy = (target.Mass*target.Vy + o.Mass*o.Vy) / o.Mass
				o.Vz = (target.Mass*target.Vz + o.Mass*o.Vz) / o.Mass
				o.Size += 1
				target.Mass = 0
				target.Stat = 2
			} else {
				//log.Println(o.Id, "crashed by", target.Id, "isTooNearly", isTooNearly, "me=", o, "ta=", target)
				target.Mass += target.Mass
				target.Vx = (target.Mass*target.Vx + o.Mass*o.Vx) / target.Mass
				target.Vy = (target.Mass*target.Vy + o.Mass*o.Vy) / target.Mass
				target.Vz = (target.Mass*target.Vz + o.Mass*o.Vz) / target.Mass
				target.Size += 1
				o.Mass = 0
				o.Stat = 2
			}
		} else {
			// 作用正常，累计计算受到的所有的万有引力
			gTmp := o.CalcGravity(&oList[i], dist)
			gAll.Ax += gTmp.Ax
			gAll.Ay += gTmp.Ay
			gAll.Az += gTmp.Az
		}
	}

	return gAll
}

// 计算天体与目标的引力
func (o *Orb) CalcGravity(target *Orb, dist float64) Acc {
	var a Acc
	// 万有引力公式
	a.A = target.Mass / (dist * dist) * G
	a.Ax = -a.A * (o.X - target.X) / dist //a.A * math.Cos(a.Dir)
	a.Ay = -a.A * (o.Y - target.Y) / dist //a.A * math.Sin(a.Dir)
	a.Az = -a.A * (o.Z - target.Z) / dist //a.A * math.Sin(a.Dir)
	return a
}

// 计算距离
func (o *Orb) CalcDist(target *Orb) float64 {
	return math.Sqrt((o.X-target.X)*(o.X-target.X) + (o.Y-target.Y)*(o.Y-target.Y) + (o.Z-target.Z)*(o.Z-target.Z))
}

// 清理orbList中的垃圾
func ClearOrbList(oList []Orb) []Orb {
	var alive int = len(oList)
	for i := 0; i < len(oList); i++ {
		if oList[i].Stat != 1 {
			oList = append(oList[:i], oList[i+1:]...)
			i--
			alive--
		}
	}
	//log.Println("when clear alive=", alive)
	return oList
}

func ShowMonitorInfo() {
	log.Println("maxVelo=", maxVeloX, maxVeloY, maxVeloZ, "maxAcc=", maxAccX, maxAccY, maxAccZ, "maxMass=", maxMassId, maxMass)
}
