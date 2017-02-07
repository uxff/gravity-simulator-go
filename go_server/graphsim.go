package main

import (
	"fmt"
	"image"
	"image/color"
	//"image/draw"
	"flag"
	//"image/jpeg"
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
	List    []int
	lastIdx int
	lastDot *WaterDot
	length  int32
	step    float64
}
type Topomap struct {
	data   []uint8
	width  int
	height int
}
type WaterMap struct {
	data   []WaterDot
	width  int
	height int
}

func (this *FlowList) Init(x int, y int, w *WaterMap, maxlen int) {
	this.List = make([]int, maxlen)
	//this.lastDot = &this.List[0]
	this.lastIdx = x + y*w.width
	this.List[0] = this.lastIdx
	this.lastDot = &w.data[this.lastIdx]
	this.lastDot.x = float32(x) + 0.5
	this.lastDot.y = float32(y) + 0.5
	this.lastDot.q = 1
	this.length = 1
	theDir := rand.Float64() * math.Pi * 2.0
	this.lastDot.dir = theDir
	this.step = 1
	fmt.Println("new dir:", theDir, this.lastDot)

	//w.data[x+y*w.width] = *this.lastDot
}

func (this *FlowList) Move2(m *Topomap, w *WaterMap) {
	ox, oy := this.lastDot.x, this.lastDot.y

	// 流入水源
	oid := int(ox) + int(oy)*m.width
	if oid < 0 || m.width*m.height < oid {
		fmt.Println("illegal oid:", oid)
		return
	}

	// lastDot indicate next
	//var nextDot = &this.List[this.length]
	var nextDot *WaterDot = nil
	//

	// 假设选择一个方向 查看是否能前进
	var allvx, allvy float64 = 0.0, 0.0
	var theDir, rollDir float64 = this.lastDot.dir, 0.0
	var tx, ty int

	var hasStep, needTurn bool = false, false
	for i := 0; i < 20; i++ {
		var pit1 float64 = rand.Float64() - rand.Float64()
		rollDir = pit1 * pit1 * pit1 * math.Pi / 2.0

		theDir += rollDir
		allvx, allvy = (math.Cos(theDir) * this.step), (math.Sin(theDir) * this.step)

		// 碰到边界
		tx, ty = int(float64(this.lastDot.x)+allvx), int(float64(this.lastDot.y)+allvy)
		if tx < 0 || ty < 0 || int(tx) > m.width-1 || int(ty) > m.height-1 {
			//fmt.Println("seems over bound tx,ty:", tx, ty, "o=", this.lastDot, "i=", i)
			continue
		}

		// 地形较高 不允许流向高处 对方海拔+对方水位-本地海拔+本地水位
		assumeFall := int(m.data[ty*m.width+tx]) - int(m.data[oid]) // + len(w.data[ty*m.width+tx].input) - len(w.data[oid].input)
		if assumeFall >= 1 {
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
				//continue
			}

		}

		hasStep = true
		//fmt.Println("got dir ok: theDir=", theDir, "rollDir=", rollDir, "target x,y,h:", tx, ty, assumeFall, "needTurn", needTurn)
		break
	}

	if hasStep {
		this.lastDot.nextIdx = tx + ty*w.width //int(nextDot.x) + int(nextDot.y)*m.width //nextDot
		nextDot = &w.data[tx+ty*w.width]
		nextDot.x, nextDot.y = this.lastDot.x, this.lastDot.y
		nextDot.x += float32(allvx)
		nextDot.y += float32(allvy)
		nextDot.dir = theDir
		nextDot.q++
		// 流入流量 应该排重
		nextDot.input = append(nextDot.input, oid)

		this.lastIdx = this.lastDot.nextIdx
		this.List[this.length] = this.lastDot.nextIdx

		// 切换指针
		this.lastDot = nextDot
		this.length++
		fmt.Println("go next ok:", nextDot, "rollDir=", rollDir, "needTurn=", needTurn, "length=", this.length)
	} else {
		//this.lastDot.q--
		fmt.Println("move failed, no step can go")
	}
}

func (this *Topomap) Init(width int, height int) {
	this.data = make([]uint8, width*height)
	this.width = width
	this.height = height
}
func (this *WaterMap) Init(width int, height int) {
	this.data = make([]WaterDot, width*height)
	this.width = width
	this.height = height
}

type Ring struct {
	x       int
	y       int
	r       int
	tiltDir float64 // 倾斜方向
	tiltLen int     // 倾斜长度
}

func colorTpl(colorTplFile string) []color.Color {
	var colorTplFileIo, _ = os.Open(colorTplFile)
	defer colorTplFileIo.Close()
	var colorTplPng, err = png.Decode(colorTplFileIo)

	if err != nil {
		fmt.Println("png.decode err:", err)
		return nil
	}
	cs := make([]color.Color, colorTplPng.Bounds().Dy())
	for i := 0; i < colorTplPng.Bounds().Dy(); i++ {
		cs[i] = colorTplPng.At(0, i)
	}
	return cs
}

func lineTo(img *image.RGBA, startX, startY, destX, destY int, c color.Color) {
	distM := math.Sqrt(float64((startX-destX)*(startX-destX) + (startY-destY)*(startY-destY)))
	var i float64
	for i = 0; i < distM/2.0; i++ {
		img.Set(startX+int(i/distM*float64(destX-startX)), startY+int(i/distM*float64(destY-startY)), c)
	}
	//img.Set(destX, destY, c)
	img.Set(startX, startY, color.RGBA{0, 0xFF, 0xFF, 0xFF})
}

func main() {
	rand.Seed(int64(time.Now().UnixNano()))

	var times = flag.Int("flow", 5, "flow move times")
	var width, height int = 500, 500
	flag.IntVar(&width, "w", width, "width of map")
	flag.IntVar(&height, "h", width, "height of map")
	var outname = flag.String("out", "testmap", "image filename of output")
	var outdir = flag.String("outdir", "output", "out put dir")
	var nHills = flag.Int("hill", 100, "hill number for making rand topo by hill")
	var hillWide = flag.Int("hill-wide", 100, "hill wide for making rand topo by hill")
	var bShowMap = flag.Bool("print", false, "print map for debug")
	var nRidge = flag.Int("ridge", 100, "ridge times for making ridge")
	var ridgeWide = flag.Int("ridge-wide", 50, "ridge wide for making ridge")
	var ridgeStep = flag.Int("ridge-step", 8, "ridge step for making ridge")
	var zoom = flag.Int("zoom", 1, "zoom of out put")

	flag.Parse()

	var m Topomap
	var w WaterMap
	var river, ridge FlowList
	w.Init(width, height)
	m.Init(width, height)
	river.Init(width/2, height/2, &w, *times+1)
	ridge.Init(width/2, height/2, &w, *nRidge+1)
	ridge.step = float64(*ridgeStep)

	if _, derr := os.Open(*outdir); derr != nil {
		fmt.Println("dir seems not exist:", *outdir, derr)
		if cerr := os.Mkdir(*outdir, os.ModePerm); cerr != nil {
			fmt.Println("os.mkdir:", *outdir, cerr)
			//return
		}
	}
	//picFile, _ := os.Create(*outname + ".jpg")
	filename := fmt.Sprintf("%s-%s", *outname, time.Now().Format("20060102150405"))
	picFile2, _ := os.Create(*outdir + "/" + filename + ".png")
	defer picFile2.Close()

	img := image.NewRGBA(image.Rect(0, 0, width**zoom, height**zoom))

	// 随机n个圆圈 累加抬高
	rings := make([]Ring, *nHills)
	for ri, _ := range rings {
		r := &rings[ri]
		r.x, r.y, r.r = (rand.Int() % width), (rand.Int() % height), (rand.Int() % (*hillWide))
		r.tiltDir, r.tiltLen = rand.Float64()*math.Pi, (rand.Int()%10)+1
	}

	// make ridge 生成ridge的痕迹
	for i := 1; i < *nRidge; i++ {
		ridge.Move2(&m, &w)
	}
	// 转换痕迹为ridge 为每个环分配随机半径
	ridgeRings := make([]Ring, ridge.length)
	for ri := 0; ri < int(ridge.length); ri++ {
		r := &ridgeRings[ri]
		r.x, r.y, r.r = ridge.List[ri]%width, ridge.List[ri]/width, (rand.Int() % (*ridgeWide))
	}
	fmt.Println("ridgeRings=", ridgeRings)

	// 生成地图 制造地形
	var tmpColor, maxColor float32 = 1, 1
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			tmpColor = 0
			// 收集ridgeRings产生的attitude
			for _, r := range ridgeRings {
				distM := (x-r.x)*(x-r.x) + (y-r.y)*(y-r.y)
				if distM <= r.r*r.r {
					//tmpColor++
					tmpColor += float32(distM) / float32(r.r*r.r) * rand.Float32()
					if maxColor < tmpColor {
						maxColor = tmpColor
					}
					//fmt.Println("color fill x,y,r,c=", x, y, r, tmpColor)
				}
			}
			// 收集rings产生的attitude
			for _, r := range rings {
				distM := (x-r.x)*(x-r.x) + (y-r.y)*(y-r.y)
				//rn := float64(r.tiltLen)*math.Sin(r.tiltDir-math.Atan2(float64(y), float64(y))) + float64(r.r)	// 尝试倾斜地图中的圆环 尝试失败
				rn := (r.r)
				if distM <= int(rn*rn) {
					//tmpColor++
					tmpColor += float32(distM) / float32(rn*rn) * rand.Float32()
					if maxColor < tmpColor {
						maxColor = tmpColor
					}
					//fmt.Println("color fill x,y,r,c=", x, y, r, tmpColor, "r=", r)
				}
			}

			m.data[x+y*width] = uint8(tmpColor) //+ uint8(rand.Int()%2) //int8(width - x)
		}
	}
	maxColor += 2

	// 获取颜色模板
	var colorTplFile string = "image/color-tpl.png"
	cs := colorTpl(colorTplFile)
	csmargin := 1
	cslen := len(cs) - csmargin
	// 地图背景地形绘制
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			tmpColor = float32(m.data[x+y*width])
			//img.Set(x, y, color.RGBA{uint8(0xFF * tmpColor / maxColor), 0xFF, uint8(0xFF * tmpColor / maxColor), 0xFF})
			// 比例上色
			//img.Set(x, y, cs[int(float32(cslen)*(1.0-tmpColor/maxColor))])
			// 按值上色
			//img.Set(x, y, cs[cslen-int(tmpColor)])
			// 放大
			for zix := 0; zix < *zoom; zix++ {
				for ziy := 0; ziy < *zoom; ziy++ {
					img.Set(x**zoom+zix, y**zoom+ziy, cs[int(float32(cslen)*(1.0-tmpColor/maxColor))])
				}
			}
		}
	}

	// 绘制flow
	fmt.Println("before move:", river.length, time.Now().UnixNano(), "maxColor=", maxColor)
	for i := 1; i < *times; i++ {
		river.Move2(&m, &w)
	}
	fmt.Println("after move:", river.length, time.Now().UnixNano())

	// 绘制river
	// 使用zoomstep lineTo
	var stepStartX, stepStartY, stepDestX, stepDestY int
	//var riverStep
	for i, dot := range river.List {
		stepStartX, stepStartY = stepDestX, stepDestY
		stepDestX, stepDestY = dot%width, dot/width
		if i == 0 || stepDestX == 0 && stepDestY == 0 {
			continue
		}
		// lineTo
		//img.Set(int(x)**zoom+*zoom/2, int(y)**zoom+*zoom/2, color.RGBA{0, 0, 0xFF, 0xFF})
		lineTo(img, int(stepStartX)**zoom+*zoom/2, int(stepStartY)**zoom+*zoom/2, int(stepDestX)**zoom+*zoom/2, int(stepDestY)**zoom+*zoom/2, color.RGBA{0, 0, 0xFF, 0xFF})
	}
	//lineTo(img, 100, 200, 200, 200, color.RGBA{0, 0, 0xFF, 0xFF})

	for i := 0; i < len(cs); i++ {
		//c := colorTplPng.At(0, i)
		c := cs[i]
		//cr, cg, cb, ca := color.RGBA()
		img.Set(0, i, c)
		img.Set(7, i, c)
		img.Set(6, i, c)
		img.Set(5, i, c)
		img.Set(4, i, c)
		img.Set(3, i, c)
		img.Set(2, i, c)
		img.Set(1, i, c)
		//fmt.Println("c:", color, i)//每个列都一样
	}

	// 输出图片文件
	//jpeg.Encode(picFile, img, nil)
	if err := png.Encode(picFile2, img); err != nil {
		fmt.Println("png.Encode error:", err)
	}

	if *bShowMap {
		//fmt.Println("map=", m)
		for di, dd := range m.data {
			fmt.Printf("%2d", dd)
			if di%width == (width - 1) {
				fmt.Printf("\n")
			}
		}
		//fmt.Println("wmap=", w)
	}
	fmt.Println("done w,h=", width, height, "maxColor=", maxColor, "nHills=", *nHills, "flowlen=", river.length, "ridgelen=", ridge.length)
}
