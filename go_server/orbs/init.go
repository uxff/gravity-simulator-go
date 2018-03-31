package orbs

import (
	"math"
	"math/rand"
)

// 配置
type InitConfig struct {
	Mass         float64
	Wide         float64
	Velo         float64
	Arrange      int     // 分布方式 0=线性 1=立方体 2=圆盘圆柱 3=球形
	Assemble     int     // 聚集方式：0=均匀分布 1=中心靠拢开方分布 2=比例加权分布 3=比例立方
	BigMass      float64 // 大块头的质量 比如处于中心的黑洞
	BigNum       int     // 大块头个数
	BigDistStyle int     // big mass orb distribute style: 0=center 1=outer edge 2=middle of one radius 3=random

}

// 初始化天体位置，质量，加速度 在一片区域随机分布
func InitOrbs(num int, config *InitConfig) []Orb {
	oList := make([]Orb, num)

	// 通用属性设置
	for i := 0; i < num; i++ {
		o := &oList[i]
		o.Mass = rand.Float64() * config.Mass
		o.Id = int32(i + 1) // rand.Int()
		//o.Stat = 1
		allMass += o.Mass
	}

	// 排列分布 与 中心聚集
	switch config.Arrange {
	case 0: //线性
		for i := 0; i < num; i++ {
			//distStep = i / distStepAll
			var wide = config.Wide
			switch config.Assemble {
			case 1:
				wide = config.Wide * math.Sqrt(float64(i+1)/float64(num))
			case 2:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 2.0)
			case 3:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 4.0)
			default:
				wide = config.Wide
			}
			o := &oList[i]
			o.X = (0.5 - rand.Float64()) * wide
			o.Y, o.Z = (0.5-rand.Float64())*config.Wide/256.0, (0.5-rand.Float64())*config.Wide/256.0

			if o.X < 0 {
				o.Vx = (1.0 + rand.Float64()) * config.Velo
				o.Vy = -(1.0 + rand.Float64()) * config.Velo //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
			} else {
				o.Vx = -(1.0 + rand.Float64()) * config.Velo
				o.Vy = (1.0 + rand.Float64()) * config.Velo
			}
			o.Vz = (rand.Float64() - 0.5) * config.Velo * 2.0 / 256.0
		}
	case 1: //立方体
		for i := 0; i < num; i++ {
			o := &oList[i]
			var wide = config.Wide
			switch config.Assemble {
			case 1:
				wide = config.Wide * math.Sqrt(float64(i+1)/float64(num))
			case 2:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 2.0)
			case 3:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 4.0)
			default:
				wide = config.Wide
			}
			o.X = (0.5 - rand.Float64()) * wide
			o.Y = (0.5 - rand.Float64()) * wide
			o.Z = (0.5 - rand.Float64()) * wide

			o.Vx = (rand.Float64() - 0.5) * config.Velo * 2.0
			o.Vy = (rand.Float64() - 0.5) * config.Velo * 2.0
			o.Vz = (rand.Float64() - 0.5) * config.Velo * 2.0
		}
	case 2: //圆盘 随机选经度 随机选半径 随机选高低 刻意降低垂直于柱面的速度
		for i := 0; i < num; i++ {
			o := &oList[i]
			long := rand.Float64() * math.Pi * 2
			high := (0.5 - rand.Float64()) * config.Wide
			var wide = config.Wide
			switch config.Assemble {
			case 1:
				wide = config.Wide * math.Sqrt(float64(i+1+MIN_CRITICAL_DIST*100)/float64(num+MIN_CRITICAL_DIST*100))
			case 2:
				wide = config.Wide * math.Pow(float64(i+1+MIN_CRITICAL_DIST*100)/float64(num+MIN_CRITICAL_DIST*100), 2.0)
			case 3:
				wide = config.Wide * math.Pow(float64(i+1+MIN_CRITICAL_DIST*100)/float64(num+MIN_CRITICAL_DIST*100), 3.0)
			case 4:
				wide = config.Wide * math.Pow(float64(i+1+MIN_CRITICAL_DIST*100)/float64(num+MIN_CRITICAL_DIST*100), 4.0)
			default:
				wide = config.Wide
			}
			radius := wide / 2.0 * math.Sqrt(rand.Float64())
			o.X, o.Y = math.Cos(long)*radius, math.Sin(long)*radius
			o.Z = high / 256.0
			//o.Vx = (rand.Float64() - 0.5) * config.Velo * 2.0 * math.Sqrt(config.Wide/(radius+1.0)) / 4.0
			//o.Vy = (rand.Float64() - 0.5) * config.Velo * 2.0 * math.Sqrt(config.Wide/(radius+1.0)) / 4.0
			o.Vx = math.Cos(long+math.Pi/2.0) * config.Velo * 2.0 //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
			o.Vy = math.Sin(long+math.Pi/2.0) * config.Velo * 2.0 //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
			o.Vz = (rand.Float64() - 0.5) * config.Velo * 2.0 / 256.0
		}
	case 3: //球形
		//方法一： 随机经度 随机半径 随机高度*sin(半径) 产生的数据从y轴上方看z面，不均匀
		//方法二： 随机经度 随机纬度=acos(rand(0-1))
		for i := 0; i < num; i++ {
			o := &oList[i]
			var wide = config.Wide
			switch config.Assemble {
			case 1:
				wide = config.Wide * math.Sqrt(float64(i+MIN_CRITICAL_DIST*100)/float64(num+MIN_CRITICAL_DIST*100))
			case 2:
				wide = config.Wide * math.Pow(float64(i+MIN_CRITICAL_DIST*100)/float64(num+MIN_CRITICAL_DIST*100), 2.0)
			case 3:
				wide = config.Wide * math.Pow(float64(i+MIN_CRITICAL_DIST*100)/float64(num+MIN_CRITICAL_DIST*100), 3.0)
			case 4:
				wide = config.Wide * math.Pow(float64(i+MIN_CRITICAL_DIST*100)/float64(num+MIN_CRITICAL_DIST*100), 4.0)
			default:
				wide = config.Wide
			}
			long := rand.Float64() * math.Pi * 2
			lati := math.Acos(rand.Float64()*2.0 - 1.0)
			radius := math.Pow(rand.Float64(), 1.0/3.0) * wide / 2.0
			o.X, o.Y = radius*math.Cos(long)*math.Sin(lati), radius*math.Sin(long)*math.Sin(lati)
			o.Z = radius * math.Cos(lati)
			o.Vx = (rand.Float64() - 0.5) * config.Velo * 2.0
			o.Vy = (rand.Float64() - 0.5) * config.Velo * 2.0
			o.Vz = (rand.Float64() - 0.5) * config.Velo * 2.0
		}
	case 4: //线性 4轴
		for i := 0; i < num; i++ {
			//distStep = i / distStepAll
			var wide = config.Wide
			switch config.Assemble {
			case 1:
				wide = config.Wide * math.Sqrt(float64(i+1)/float64(num))
			case 2:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 2.0)
			case 3:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 4.0)
			default:
				wide = config.Wide
			}
			o := &oList[i]

			o.X, o.Y, o.Z = (0.5-rand.Float64())*config.Wide/256.0, (0.5-rand.Float64())*config.Wide/256.0, (0.5-rand.Float64())*config.Wide/256.0
			o.Vx, o.Vy, o.Vz = (rand.Float64()-0.5)*config.Velo/256.0, (rand.Float64()-0.5)*config.Velo/256.0, (rand.Float64()-0.5)*config.Velo/256.0
			award := i % 2

			switch award {
			case 0:
				o.X = (0.5 - rand.Float64()) * wide
				if o.X < 0 {
					o.Vx = (1.0 + rand.Float64()) * config.Velo
					o.Vy = -(1.0 + rand.Float64()) * config.Velo //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
				} else {
					o.Vx = -(1.0 + rand.Float64()) * config.Velo
					o.Vy = (1.0 + rand.Float64()) * config.Velo
				}
			case 1:
				o.Y = (0.5 - rand.Float64()) * wide
				if o.Y < 0 {
					o.Vy = (1.0 + rand.Float64()) * config.Velo
					o.Vz = -(1.0 + rand.Float64()) * config.Velo //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
				} else {
					o.Vy = -(1.0 + rand.Float64()) * config.Velo
					o.Vz = (1.0 + rand.Float64()) * config.Velo
				}
			case 2:
				o.Z = (0.5 - rand.Float64()) * wide
				if o.Z < 0 {
					o.Vz = (1.0 + rand.Float64()) * config.Velo
					o.Vx = -(1.0 + rand.Float64()) * config.Velo //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
				} else {
					o.Vz = -(1.0 + rand.Float64()) * config.Velo
					o.Vx = (1.0 + rand.Float64()) * config.Velo
				}
			default:
			}
		}
	case 5: //线性 6轴
		for i := 0; i < num; i++ {
			//distStep = i / distStepAll
			var wide = config.Wide
			switch config.Assemble {
			case 1:
				wide = config.Wide * math.Sqrt(float64(i+1)/float64(num))
			case 2:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 2.0)
			case 3:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 4.0)
			default:
				wide = config.Wide
			}
			o := &oList[i]

			o.X, o.Y, o.Z = (0.5-rand.Float64())*config.Wide/256.0, (0.5-rand.Float64())*config.Wide/256.0, (0.5-rand.Float64())*config.Wide/256.0
			o.Vx, o.Vy, o.Vz = (rand.Float64()-0.5)*config.Velo/256.0, (rand.Float64()-0.5)*config.Velo/256.0, (rand.Float64()-0.5)*config.Velo/256.0
			award := i % 3

			switch award {
			case 0:
				o.X = (0.5 - rand.Float64()) * wide
				if o.X < 0 {
					o.Vx = (1.0 + rand.Float64()) * config.Velo
					o.Vy = -(1.0 + rand.Float64()) * config.Velo //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
				} else {
					o.Vx = -(1.0 + rand.Float64()) * config.Velo
					o.Vy = (1.0 + rand.Float64()) * config.Velo
				}
			case 1:
				o.Y = (0.5 - rand.Float64()) * wide
				if o.Y < 0 {
					o.Vy = (1.0 + rand.Float64()) * config.Velo
					o.Vz = -(1.0 + rand.Float64()) * config.Velo //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
				} else {
					o.Vy = -(1.0 + rand.Float64()) * config.Velo
					o.Vz = (1.0 + rand.Float64()) * config.Velo
				}
			case 2:
				o.Z = (0.5 - rand.Float64()) * wide
				if o.Z < 0 {
					o.Vz = (1.0 + rand.Float64()) * config.Velo
					o.Vx = -(1.0 + rand.Float64()) * config.Velo //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
				} else {
					o.Vz = -(1.0 + rand.Float64()) * config.Velo
					o.Vx = (1.0 + rand.Float64()) * config.Velo
				}
			default:
			}
		}
	case 6: //线性 1轴
		for i := 0; i < num; i++ {
			//distStep = i / distStepAll
			var wide = config.Wide
			switch config.Assemble {
			case 1:
				wide = config.Wide * math.Sqrt(float64(i+1)/float64(num))
			case 2:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 2.0)
			case 3:
				wide = config.Wide * math.Pow(float64(i+1)/float64(num), 4.0)
			default:
				wide = config.Wide
			}
			o := &oList[i]
			o.X = (rand.Float64()) * wide
			o.Y, o.Z = (0.5-rand.Float64())*config.Wide/256.0, (0.5-rand.Float64())*config.Wide/256.0

			if o.X < 0 {
				o.Vx = (1.0 + rand.Float64()) * config.Velo
				o.Vy = -(1.0 + rand.Float64()) * config.Velo //* math.Sqrt(config.Wide/(radius+1.0)) / 4.0
			} else {
				o.Vx = -(1.0 + rand.Float64()) * config.Velo
				o.Vy = (1.0 + rand.Float64()) * config.Velo
			}
			o.Vz = (rand.Float64() - 0.5) * config.Velo * 2.0 / 256.0
		}
	default:
	}

	// 如果配置了大块头质量 0=中心 1=边缘 2=半径的中点 3=随机
	if config.BigMass != 0.0 {
		for i := 0; i < config.BigNum && config.BigNum <= len(oList); i++ {

			eternalOrb := &oList[num-1-i]
			allMass += config.BigMass - eternalOrb.Mass
			eternalOrb.Mass = config.BigMass
			eternalOrb.X, eternalOrb.Y, eternalOrb.Z = 0, 0, 0
			eternalOrb.Vx, eternalOrb.Vy, eternalOrb.Vz = 0, 0, 0
			switch config.BigDistStyle {
			case 1:
				// 环形分布
				eternalOrb.X = math.Cos(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Wide / 2.0
				eternalOrb.Y = math.Sin(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Wide / 2.0
				// 逆时针运动
				eternalOrb.Vx = -math.Sin(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Velo
				eternalOrb.Vy = math.Cos(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Velo
			case 2:
				eternalOrb.X = math.Cos(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Wide / 2.0 / 2.0
				eternalOrb.Y = math.Sin(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Wide / 2.0 / 2.0
				eternalOrb.Vx = -math.Sin(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Velo
				eternalOrb.Vy = math.Cos(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Velo
			case 3:
				eternalOrb.X = math.Cos(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Wide / 2.0 * rand.Float64()
				eternalOrb.Y = math.Sin(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Wide / 2.0 * rand.Float64()
				eternalOrb.Vx = -math.Sin(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Velo
				eternalOrb.Vy = math.Cos(float64(i)*math.Pi*2/float64(config.BigNum)) * config.Velo
			case 0:
				fallthrough
			default:
			}
		}

	}
	return oList
}
