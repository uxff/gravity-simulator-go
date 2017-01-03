package main

import (
	"fmt"
	"image"
	"image/color"
	//"image/draw"
	"flag"
	"image/jpeg"
	"image/png"
	"math"
	"math/rand"
	"os"
	"time"
)

type WaterDot struct {
	x       float32
	y       float32
	dir     float64
	nextIdx int
	//h       int8 // 高度
	q     int   // 流量 0=无
	input []int // 流入坐标
}
type FlowList struct {
	List    []WaterDot
	lastIdx int
	lastDot *WaterDot
	length  int32
}
type Topomap struct {
	data   []int8
	width  int
	height int
}
type WaterMap struct {
	data   []WaterDot
	width  int
	height int
}

func (this *FlowList) Init(x int, y int, w *WaterMap) {
	this.List = make([]WaterDot, 500)
	this.lastDot = &this.List[0]
	this.lastDot.x = float32(x) + 0.5
	this.lastDot.y = float32(y) + 0.5
	this.lastDot.q = 1
	this.length = 1
	theDir := rand.Float64() * math.Pi * 2.0
	this.lastDot.dir = theDir
	fmt.Println("new dir:", theDir, this.lastDot)

	this.lastIdx = x + y*w.width
	w.data[x+y*w.width] = *this.lastDot
}

func (this *FlowList) Move2(m *Topomap, w *WaterMap) {
	ox, oy := this.lastDot.x, this.lastDot.y

	oid := int(ox) + int(oy)*m.width
	if oid < 0 || m.width*m.height < oid {
		fmt.Println("illegal oid:", oid)
		return
	}

	// lastDot indicate next
	var nextDot = &this.List[this.length]
	nextDot.x, nextDot.y = this.lastDot.x, this.lastDot.y

	// 假设选择一个方向 查看是否能前进
	var allvx, allvy float64 = 0.0, 0.0
	var theDir, rollDir float64 = this.lastDot.dir, 0.0

	var hasStep, needTurn bool = false, false
	for i := 0; i < 20; i++ {
		var pit1 float64 = rand.Float64() - rand.Float64()
		rollDir = pit1 * pit1 * pit1 * math.Pi / 2.0

		theDir += rollDir
		allvx, allvy = (math.Cos(theDir)), (math.Sin(theDir))

		// 碰到边界
		tx, ty := int(float64(this.lastDot.x)+allvx), int(float64(this.lastDot.y)+allvy)
		if tx < 0 || ty < 0 || int(tx) > m.width-1 || int(ty) > m.height-1 {
			fmt.Println("seems over bound tx,ty:", tx, ty, "o=", this.lastDot, "i=", i)
			continue
		}

		// 地形较高
		assumeFall := m.data[ty*m.width+tx] - m.data[oid]
		if assumeFall > 1 {
			fmt.Println("seems flow up, fall=", assumeFall, allvx, allvy, "i=", i)
			continue
		}

		// 碰到自己 碰到水域
		if w.data[tx+ty*w.width].q >= this.lastDot.q {
			//qFall := w.data[tx+ty*w.width].q - this.lastDot.q
			//fmt.Println("seems turn self, qFall:", qFall, "i=", i)
			// 如果流入的流量小于流出的流量 则死循环
			var prepreFall int = 0
			for _, dotidx := range w.data[tx+ty*w.width].input {
				if dotidx == oid {
					//fmt.Println("do not count self:", dotidx)
					continue
				}
				prepreFall += w.data[dotidx].q
			}
			if w.data[tx+ty*w.width].q >= prepreFall+1 {
				fmt.Println("seemes turn self, too quantity", w.data[tx+ty*w.width], "oid=", oid, "i=", i, "length=", this.length)
				needTurn = true
				continue
			}

		}

		hasStep = true
		//fmt.Println("got dir ok: theDir=", theDir, "rollDir=", rollDir, "target x,y,h:", tx, ty, assumeFall, "needTurn", needTurn)
		break
	}

	if hasStep {
		nextDot.x += float32(allvx) // nextDot.vx
		nextDot.y += float32(allvy) // nextDot.vy
		nextDot.dir = theDir
		nextDot.q++
		this.lastDot.nextIdx = int(nextDot.x) + int(nextDot.y)*m.width //nextDot
		nextDot.input = append(nextDot.input, oid)

		this.lastIdx = this.lastDot.nextIdx
		w.data[this.lastDot.nextIdx].q++
		w.data[this.lastDot.nextIdx].dir = theDir
		w.data[this.lastDot.nextIdx].input = append(w.data[this.lastDot.nextIdx].input, this.lastDot.nextIdx)

		// 切换指针
		this.lastDot = nextDot
		this.length++
		fmt.Println("go next ok:", nextDot, "rollDir=", rollDir, "needTurn", needTurn, "length=", this.length)
	} else {
		//this.lastDot.q--
		fmt.Println("move failed, no step can go")
	}
}

func (this *Topomap) Init(width int, height int) {
	this.data = make([]int8, width*height)
	this.width = width
	this.height = height
}
func (this *WaterMap) Init(width int, height int) {
	this.data = make([]WaterDot, width*height)
	this.width = width
	this.height = height
}

type Ring struct {
	x int
	y int
	r int
}

func main() {
	rand.Seed(int64(time.Now().UnixNano()))

	var times = flag.Int("times", 5, "move times")
	flag.Parse()

	var m Topomap
	var w WaterMap
	var width, height int = 500, 500
	w.Init(width, height)
	m.Init(width, height)
	var river FlowList
	river.Init(width/2, height/2, &w)

	picFile, _ := os.Create("testmap.jpg")
	picFile2, _ := os.Create("testmap.png")
	defer picFile.Close()

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 随机n个圆圈 累加抬高
	n := 40
	rings := make([]Ring, n)
	for ri, _ := range rings {
		r := &rings[ri]
		r.x, r.y, r.r = (rand.Int()%width/2)+width/4, (rand.Int()%height/2)+height/4, (rand.Int()%width)/4
	}
	fmt.Println("rings=", rings)

	// 生成地图 绘制地图
	var tmpColor, maxColor uint8 = 1, 1
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// 方格子地图
			if y%5 == 0 {
				if x%5 == 0 {
					// 随机
					//tmpColor = int8(rand.Int() % 127)
					// 随y越来越高
					//tmpColor = int8(y * 255 / width)
					// 由中心朝四周越来越低
					//tmpColor = 127 - int8(math.Sqrt(float64((x-width/2)*(x-width/2)+(y-width/2)*(y-width/2)))/float64(width)*127)
					// 由中心朝四周越来越低 并随机增减
					//tmpColor = 127 - int8(math.Sqrt(float64((x-width/2)*(x-width/2)+(y-width/2)*(y-width/2)))/float64(width)*123) - int8(rand.Int()%3)
				}
			} else {
				//tmpColor = m.data[x+y*width-width]
			}
			tmpColor = 0
			for _, r := range rings {
				if (x-r.x)*(x-r.x)+(y-r.y)*(y-r.y) < r.r*r.r {
					tmpColor++
					if maxColor < tmpColor {
						maxColor = tmpColor
					}
					//fmt.Println("color fill x,y,r,c=", x, y, r, tmpColor)
				}
			}

			m.data[x+y*width] = int8(tmpColor) //int8(width - x)
			img.Set(x, y, color.RGBA{uint8(tmpColor * 10), 0xFF, uint8(tmpColor * 10), 0xFF})
		}
	}

	// 绘制flow
	fmt.Println("before move:", river, time.Now().UnixNano(), "maxColor=", maxColor)
	for i := 1; i < *times; i++ {
		river.Move2(&m, &w)
	}
	fmt.Println("after move:", river, time.Now().UnixNano())

	// 绘制river
	for _, dot := range river.List {
		img.Set(int(dot.x), int(dot.y), color.RGBA{0, 0, 0xFF, 0xFF})
	}

	jpeg.Encode(picFile, img, nil)
	png.Encode(picFile2, img)

	fmt.Println("done", math.Sin(1.0), "maxColor", maxColor)
}
