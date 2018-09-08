/*
	usage: time ./topomaker -w 800 -h 800 -hill 200 -hill-wide 200 -ridge 2 -ridge-wide 50 -times 1000 -dropnum 100 -zoom 5
    todo: table lize with http server
*/
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	drawer "./drawer"
)

// 使用水滴滚动
type Droplet struct {
	x         float32
	y         float32
	fallPower int     // 落差能量
	vx        float32 // 滑行速度
	vy        float32
	hisway    []int
}

// 将变成固定不移动
type WaterDot struct {
	x       float32 // 将不变化 =Topomap[x,y] +(0.5, 0.5)
	y       float32
	xPower  float32 // v2 根据地形得出 初始化后不变(地形改变则会变) 基于Atan2 范围(-1,1)
	yPower  float32 // v2 根据地形得出 初始化后不变(地形改变则会变)
	h       int     // 积水高度，产生积水不参与流动，流动停止 // v2将由水滴实体代替该变量
	q       int     // 流量 0=无 历史流量    //
	xPowerQ float32 // v2 根据周围流量算出 每次update变化 基于Atan2 范围(-1,1)
	yPowerQ float32 // v2
}
type Topomap struct {
	data   []uint8 // 对应坐标只保存高度
	width  int
	height int
}
type WaterMap struct {
	data   []WaterDot
	width  int
	height int
}

// 先处理场向量 // 再注水流动

// 预先处理每个点的场向量  只计算地势的影响，不考虑流量的影响
// 假设每个点都有一个场，计算出这个场的方向
// 启动只执行1次
// @param Topomap m is basic topomap
// @param int ring 表示计算到几环 默认2环
func (w *WaterMap) AssignVector(m *Topomap, ring int) {
	for idx, curDot := range w.data {
		var xPower, yPower int // xPower, yPower 单位为1
		// 2nd ring
		_, lowestPos := curDot.getLowestNeighbors(curDot.getNeighbors(w), m)
		for _, neiPos := range lowestPos {
			xPower += 4 * (neiPos.x - int(idx%w.width))
			yPower += 4 * (neiPos.y - int(idx/w.width))
		}

		// 3rd ring. done 三环的影响力是二环的1/4
		if ring >= 3 {
			_, lowestPos = curDot.getLowestNeighbors(curDot.get3rdNeighbors(w), m)
			for _, neiPos := range lowestPos {
				xPower += neiPos.x - int(idx%w.width)
				yPower += neiPos.y - int(idx/w.width)
			}
		}

		// 四环 四环影响力是二环的1/16 暂不实现4环

		if xPower != 0 || yPower != 0 {
			thedir := math.Atan2(float64(yPower), float64(xPower))
			w.data[idx].xPower, w.data[idx].yPower = float32(math.Cos(thedir)), float32(math.Sin(thedir))
		}
	}
}

// 按照周围流量更新场向量
// powerRate 一般指定小于1 比如0.1
func (w *WaterMap) UpdateVectorByQuantity(m *Topomap, ring int, powerRate float32) {
	for idx, curDot := range w.data {
		//go func() {
		var xPower, yPower int // xPower, yPower 单位为1
		// 2nd ring
		_, mostQuanPos := curDot.getPostQuanNeighbors(curDot.getNeighbors(w), w)
		for _, neiPos := range mostQuanPos {
			xPower += 4 * (neiPos.x - int(idx%w.width))
			yPower += 4 * (neiPos.y - int(idx/w.width))
		}

		// 3rd ring. done 三环的影响力是二环的1/4
		if ring >= 3 {
			_, mostQuanPos = curDot.getPostQuanNeighbors(curDot.get3rdNeighbors(w), w)
			for _, neiPos := range mostQuanPos {
				xPower += neiPos.x - int(idx%w.width)
				yPower += neiPos.y - int(idx/w.width)
			}
		}

		if xPower != 0 || yPower != 0 {
			thedir := math.Atan2(float64(yPower), float64(xPower))
			w.data[idx].xPowerQ, w.data[idx].yPowerQ = float32(math.Cos(thedir))*powerRate, float32(math.Sin(thedir))*powerRate
		}
		//}() //可以不等待 //使用go反而慢
	}
}

func UpdateDroplets(times int, drops []*Droplet, m *Topomap, w *WaterMap) []*Droplet {
	for i := 1; i <= times; i++ {
		wg := &sync.WaitGroup{}
		for _, d := range drops {
			wg.Add(1)
			go func(d *Droplet) {
				d.Move(m, w)
				wg.Done()
			}(d)
		}

		wg.Wait()
		if i%100 == 0 {
			w.UpdateVectorByQuantity(m, 2, 0.1)
			drops = ClearDroplets(drops)
			log.Printf("drops cleard len=%d", len(drops))
		}
	}
	return drops
}

func ClearDroplets(drops []*Droplet) []*Droplet {
	newDrops := make([]*Droplet, 0)
	for idx, d := range drops {
		if d.vx == 0 && d.vy == 0 {
			log.Printf("clear:[%d]=%+v", idx, *d)
			//drops = append(drops[:idx], drops[idx+1:]...)//panic
			//idx--
			continue
		}
		newDrops = append(newDrops, d)
	}
	return newDrops
}

func MakeDroplet(w *WaterMap) *Droplet {
	idx := rand.Int() % len(w.data)
	d := &Droplet{
		x:      float32(idx%w.width) + 0.5,
		y:      float32(idx/w.width) + 0.5,
		hisway: []int{idx},
	}

	w.data[idx].h++
	//w.data[idx].q++ //初次不算流量
	return d
}

func (d *Droplet) Move(m *Topomap, w *WaterMap) {
	oldIdx := int(d.x) + int(d.y)*w.width
	if oldIdx >= len(w.data) {
		log.Printf("oldIdx(%d) out of data. stop it.", oldIdx)
		return
	}

	// 没有场 可撒欢
	if w.data[oldIdx].xPower == 0 && w.data[oldIdx].yPower == 0 {
		//log.Printf("no field power, try slip(x=%f,y=%f)", d.x, d.y)
		d.MoveByFallPower(m, w)
		return
	}

	// 场速度与自身速度的平均值
	tmpX := d.x + (w.data[oldIdx].xPower+d.vx)/2 // todo:精度损失风险
	tmpY := d.y + (w.data[oldIdx].yPower+d.vy)/2

	// 越界判断
	if int(tmpX) < 0 || int(tmpX) > w.width-1 || int(tmpY) < 0 || int(tmpY) > w.height-1 {
		log.Printf("droplet move out of bound(x=%f,y=%f). stop move.", tmpX, tmpY)
		return
	}

	newIdx := int(tmpX) + int(tmpY)*w.width
	// 无力场，待在原地
	if newIdx == oldIdx {
		//log.Printf("no field power. stay here.")
		return
	}

	if newIdx >= w.width*w.height {
		log.Printf("newIdx(%d) out of data range, ignore", newIdx)
		return
	}

	mu := sync.Mutex{}
	mu.Lock()
	defer mu.Unlock()

	// droplet no need lock
	d.x, d.y = tmpX, tmpY
	d.hisway = append(d.hisway, newIdx)
	d.vx, d.vy = w.data[oldIdx].xPower, w.data[oldIdx].yPower
	d.fallPower += int(m.data[oldIdx]-m.data[newIdx]) * 100

	w.data[oldIdx].h--
	w.data[newIdx].h++
	w.data[oldIdx].q++ // 流出，才算流量
}

// 根据落差能量移动 类似滑行 slip todo:浮动(撒欢)
func (d *Droplet) MoveByFallPower(m *Topomap, w *WaterMap) {
	mu := sync.Mutex{}
	mu.Lock()
	defer mu.Unlock()

	if d.fallPower > 0 {
		oldIdx := int(d.x) + int(d.y)*w.width
		if oldIdx >= len(w.data) {
			log.Printf("oldIdx(%d) out of data. stop it.", oldIdx)
			return
		}

		tmpDir := rand.Float64() - rand.Float64()
		fx, fy := float32(math.Cos(tmpDir)), float32(math.Sin(tmpDir))
		//d.vx, d.vy = d.vx+fx/2.0, d.vy+fy/2.0
		tmpX := d.x + d.vx + fx/2.0 // todo:精度损失风险
		tmpY := d.y + d.vy + fy/2.0

		// 越界判断
		if int(tmpX) < 0 || int(tmpX) > w.width-1 || int(tmpY) < 0 || int(tmpY) > w.height-1 {
			log.Printf("droplet slip out of bound(x=%f,y=%f). stop move.", tmpX, tmpY)
			return
		}

		newIdx := int(tmpX) + int(tmpY)*w.width
		// 无力场，待在原地
		if newIdx == oldIdx {
			//log.Printf("no field power. stay here.")
			return
		}

		if newIdx > w.width*w.height {
			log.Printf("newIdx(%d) out of data range, ignore", newIdx)
			return
		}

		// droplet no need lock
		d.x, d.y = tmpX, tmpY
		d.hisway = append(d.hisway, newIdx)
		d.fallPower--

		w.data[oldIdx].h--
		w.data[newIdx].h++
		w.data[oldIdx].q++ // 流出，才算流量
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
	// 把点的实际基点摆在中间
	for x := 0; x < this.width; x++ {
		for y := 0; y < this.height; y++ {
			this.data[x+y*this.width].x = float32(x) + 0.5
			this.data[x+y*this.width].y = float32(y) + 0.5
		}
	}
}

type Hill struct {
	x       int
	y       int
	r       int
	h       int
	tiltDir float64 // 倾斜方向
	tiltLen int     // 倾斜长度
}

/*返回颜色数组，下标越大颜色海拔越高*/
func colorTpl(colorTplFile string) []color.Color {
	var colorTplFileIo, _ = os.Open(colorTplFile)
	defer colorTplFileIo.Close()
	var colorTplPng, err = png.Decode(colorTplFileIo)

	if err != nil {
		log.Println("png.decode err when read colorTpl:", err)
		return nil
	}
	theLen := colorTplPng.Bounds().Dy()
	cs := make([]color.Color, theLen)
	for i := 0; i < theLen; i++ {
		cs[i] = colorTplPng.At(0, theLen-i-1)
	}
	return cs
}

func lineTo(img *image.RGBA, startX, startY, destX, destY int, lineColor, startColor color.Color, scale float64) {
	distM := math.Sqrt(float64((startX-destX)*(startX-destX) + (startY-destY)*(startY-destY)))
	var i float64
	for i = 0; i < distM*scale; i++ {
		img.Set(startX+int(i/distM*float64(destX-startX)), startY+int(i/distM*float64(destY-startY)), lineColor)
	}
	// 线段最后一点 绘制成始发地地形的颜色 startColor
	if startX != destX && startY != destY {
		img.Set(startX+int(i/distM*float64(destX-startX)), startY+int(i/distM*float64(destY-startY)), startColor)
	}
}

// 此函数固定返回本坐标周边2环8个边界点，可能包含超出地图边界的点
func (d *WaterDot) getNeighbors(w *WaterMap) []struct{ x, y int } {
	pos := make([]struct{ x, y int }, 8)
	pos[0].x, pos[0].y = int(d.x+1), int(d.y)
	pos[1].x, pos[1].y = int(d.x+1), int(d.y-1)
	pos[2].x, pos[2].y = int(d.x), int(d.y-1)
	pos[3].x, pos[3].y = int(d.x-1), int(d.y-1)
	pos[4].x, pos[4].y = int(d.x-1), int(d.y)
	pos[5].x, pos[5].y = int(d.x-1), int(d.y+1)
	pos[6].x, pos[6].y = int(d.x), int(d.y+1)
	pos[7].x, pos[7].y = int(d.x+1), int(d.y+1)
	return pos
}

// 此函数固定返回本点周边3环12个边界点，可能包含超出地图边界的点
func (d *WaterDot) get3rdNeighbors(w *WaterMap) []struct{ x, y int } {
	pos := make([]struct{ x, y int }, 12)
	// right
	pos[0].x, pos[0].y = int(d.x+2), int(d.y)
	pos[1].x, pos[1].y = int(d.x+2), int(d.y-1)
	// top
	pos[2].x, pos[2].y = int(d.x+1), int(d.y-2)
	pos[3].x, pos[3].y = int(d.x), int(d.y-2)
	pos[4].x, pos[4].y = int(d.x-1), int(d.y-2)
	// left
	pos[5].x, pos[5].y = int(d.x-2), int(d.y-1)
	pos[6].x, pos[6].y = int(d.x-2), int(d.y)
	pos[7].x, pos[7].y = int(d.x-2), int(d.y+1)
	// bottom
	pos[8].x, pos[8].y = int(d.x-1), int(d.y+2)
	pos[9].x, pos[9].y = int(d.x), int(d.y+2)
	pos[10].x, pos[10].y = int(d.x+1), int(d.y+2)
	// right
	pos[11].x, pos[11].y = int(d.x+2), int(d.y+1)
	return pos
}

/*获取周围最低的点 最低点集合数组中随机取一个 返回安全的坐标，不在地图外*/
func (d *WaterDot) getLowestNeighbors(arrNei []struct{ x, y int }, m *Topomap) (lowestLevel int, lowestPos []struct{ x, y int }) {
	// 原理： highMap[high] = []struct{int,int}
	highMap := make(map[int][]struct{ x, y int }, 8)
	for _, nei := range arrNei {
		if nei.x < 0 || nei.x > m.width-1 || nei.y < 0 || nei.y > m.height-1 {
			// 超出地图边界的点
			continue
		}
		// 邻居的高度 todo: 有BUG 此处不能加本地的水位 要加邻居的水位
		high := int(m.data[nei.x+nei.y*m.width]) + d.h
		if len(highMap[int(high)]) == 0 {
			highMap[high] = []struct{ x, y int }{{nei.x, nei.y}} //make([]struct{ x, y int }, 1)
			//highMap[int(m.data[nei.x+nei.y*w.width])][0].x, highMap[int(m.data[nei.x+nei.y*w.width])][0].y = nei.x, nei.y
		} else {
			highMap[high] = append(highMap[high], struct{ x, y int }{nei.x, nei.y})
		}
	}
	lowestLevel = 1000
	for k, _ := range highMap {
		if k < lowestLevel {
			lowestLevel = k
		}
	}
	//log.Println("lowest,highMap,count(highMap),d=", lowest, highMap, len(highMap), *d)
	if len(highMap[lowestLevel]) == 0 {
		log.Printf("how is can be zero?")
		return lowestLevel, nil
	}
	return lowestLevel, highMap[lowestLevel]
}

/*获取周围流量最大的点 流量最大的点集合数组中随机取一个 返回安全的坐标，不在地图外*/
func (d *WaterDot) getPostQuanNeighbors(arrNei []struct{ x, y int }, w *WaterMap) (mostQuanLevel int, poses []struct{ x, y int }) {
	// 原理： highMap[quantity] = []struct{int,int}
	highMap := make(map[int][]struct{ x, y int }, 8)
	for _, nei := range arrNei {
		if nei.x < 0 || nei.x > w.width-1 || nei.y < 0 || nei.y > w.height-1 {
			// 超出地图边界的点
			continue
		}
		high := int(w.data[nei.x+nei.y*w.width].q)
		if len(highMap[int(high)]) == 0 {
			highMap[high] = []struct{ x, y int }{{nei.x, nei.y}} //make([]struct{ x, y int }, 1)
			//highMap[int(m.data[nei.x+nei.y*w.width])][0].x, highMap[int(m.data[nei.x+nei.y*w.width])][0].y = nei.x, nei.y
		} else {
			highMap[high] = append(highMap[high], struct{ x, y int }{nei.x, nei.y})
		}
	}
	mostQuanLevel = 0
	for k, _ := range highMap {
		if k > mostQuanLevel {
			mostQuanLevel = k
		}
	}
	//log.Println("lowest,highMap,count(highMap),d=", lowest, highMap, len(highMap), *d)
	if len(highMap[mostQuanLevel]) == 0 {
		log.Printf("how is can be zero?")
		return mostQuanLevel, nil
	}
	return mostQuanLevel, highMap[mostQuanLevel]
}

func main() {
	rand.Seed(int64(time.Now().UnixNano()))

	var width, height int = 500, 500
	flag.IntVar(&width, "w", width, "width of map")
	flag.IntVar(&height, "h", width, "height of map")
	var outname = flag.String("out", "testmap", "image filename of output")
	var outdir = flag.String("outdir", "output", "out put dir")
	var nHills = flag.Int("hill", 100, "hill number for making rand topo by hill")
	var hillWide = flag.Int("hill-wide", 100, "hill wide for making rand topo by hill")
	var bShowMap = flag.Bool("print", false, "print map for debug")
	var nRidge = flag.Int("ridge", 1, "num of ridges for making ridges")
	var ridgeWide = flag.Int("ridge-wide", 50, "ridge wide for making ridge each")
	//var ridgeStep = flag.Int("ridge-step", 8, "ridge step for making ridge each")
	var ridgeLen = flag.Int("ridge-len", 100, "ridge length when making ridge each")
	var dropNum = flag.Int("dropnum", 100, "number of drops")
	var times = flag.Int("times", 1000, "update times")
	var zoom = flag.Int("zoom", 1, "zoom of out put image")
	var addr = flag.String("addr", "", "addr of http server to listen and to show img on html(deprecated)")
	var riverArrowScale = flag.Float64("river-arrow-scale", 0.8, "river arrow scale")

	flag.Parse()

	var m Topomap
	var w WaterMap

	// 初始化 watermap topomap
	w.Init(width, height)
	m.Init(width, height)

	if _, derr := os.Open(*outdir); derr != nil {
		log.Println("output dir seems not exist:", *outdir, derr)
		if cerr := os.Mkdir(*outdir, os.ModePerm); cerr != nil {
			log.Println("os.mkdir:", *outdir, cerr)
		}
	}

	// 随机n个圆圈 累加抬高 输出到m中
	hills := make([]Hill, *nHills)
	for ri, _ := range hills {
		r := &hills[ri]
		r.x, r.y, r.r, r.h = (rand.Int() % width), (rand.Int() % height), (rand.Int()%(*hillWide) + 1), (rand.Int()%(5) + 2)
		r.tiltDir, r.tiltLen = rand.Float64()*math.Pi, (rand.Int()%20)+1
	}

	// 转换痕迹为ridge 为每个环分配随机半径 输出到m中
	var ridgeHills []Hill
	for ri := 0; ri < *nRidge; ri++ {
		ridgeHills = append(ridgeHills, MakeRidge(*ridgeLen, *ridgeWide, width, height)...)
	}
	//log.Println("ridgeHills=", ridgeHills)

	// 生成地图 制造地形 将上面生成的ridge和hills输出到m上
	var tmpColor, maxColor float32 = 1, 1
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			tmpColor = 0
			// 收集ridgeHills产生的altitude
			for _, r := range ridgeHills {
				distM := (x-r.x)*(x-r.x) + (y-r.y)*(y-r.y)
				rn := (r.r)
				if distM <= r.r*r.r {
					//tmpColor++
					tmpColor += float32(r.h) - float32(float64(r.h)*math.Sqrt(math.Sqrt(float64(distM)/float64((rn*rn)))))
					//tmpColor += float32(distM) / float32(r.r*r.r) * rand.Float32()
					if maxColor < tmpColor {
						maxColor = tmpColor
					}
					//log.Println("color fill x,y,r,c=", x, y, r, tmpColor)
				}
			}
			// 收集hills产生的attitude
			for _, r := range hills {
				distM := (x-r.x)*(x-r.x) + (y-r.y)*(y-r.y)
				//rn := float64(r.tiltLen)*math.Sin(r.tiltDir-math.Atan2(float64(y), float64(x))) + float64(r.r)	// 尝试倾斜地图中的圆环 尝试失败
				rn := (r.r)
				if distM <= int(rn*rn) {
					// 产生的ring中间隆起
					tmpColor += float32(r.h) - float32(float64(r.h)*math.Sqrt(math.Sqrt(float64(distM)/float64((rn*rn)))))
					//tmpColor += float32(distM) / float32(rn*rn) * rand.Float32()
					if maxColor < tmpColor {
						maxColor = tmpColor
					}
					//log.Println("color fill x,y,r,c=", x, y, r, tmpColor, "r=", r)
				}
			}

			m.data[x+y*width] = uint8(tmpColor) //+ uint8(rand.Int()%2) //int8(width - x)
		}
	}
	maxColor *= 1.5

	w.AssignVector(&m, 3)

	// 生成一组随机*Droplet
	drops := make([]*Droplet, *dropNum)
	for di := 0; di < *dropNum; di++ {
		drops[di] = MakeDroplet(&w)
	}

	drops = UpdateDroplets(*times, drops, &m, &w)
	log.Printf("update done. times=%d num drops=%d->%d", *times, *dropNum, len(drops))

	// then draw
	img := image.NewRGBA(image.Rect(0, 0, width**zoom, height**zoom))

	DrawToImg(img, &m, &w, maxColor, *zoom, *riverArrowScale, drops)

	wgm := sync.WaitGroup{}
	if *addr != "" {
		wgm.Add(2)
		go func() { drawer.StartHtmlDrawer(*addr); wgm.Done() }()
		go func() { DrawToHtml(&w, &m); wgm.Done() }()
		log.Printf("drow to html ok, open host(%s) and view", *addr)
	}

	// 输出图片文件
	wgm.Add(1)
	go func() {
		ImgToFile(fmt.Sprintf("%s/%s-%s.png", *outdir, *outname, time.Now().Format("20060102150405")), img, "png")
		wgm.Done()
	}()

	// 如果需要控制台打印地形
	if *bShowMap {
		wgm.Add(1)
		go func() { DrawToConsole(&m); wgm.Done() }()
	}
	wgm.Wait()
	log.Println("done w,h=", width, height, "maxColor=", maxColor, "nHills=", *nHills, "nRidge=", *nRidge, "ridgelen=", *ridgeLen)
	for di, d := range drops {
		log.Printf("[%d]=%+v", di, *d)
	}

}

func DrawToImg(img *image.RGBA, m *Topomap, w *WaterMap, maxColor float32, zoom int, riverArrowScale float64, drops []*Droplet) {
	height := m.height
	width := m.width
	var tmpColor float32 = 1

	// 获取颜色模板
	var colorTplFile string = "image/color-tpl.png"
	cs := colorTpl(colorTplFile)
	cslen := len(cs) - 1
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
			for zix := 0; zix < zoom; zix++ {
				for ziy := 0; ziy < zoom; ziy++ {
					img.Set(x*zoom+zix, y*zoom+ziy, cs[int(float32(cslen)*(tmpColor/maxColor))])
				}
			}
		}
	}
	// 绘制WaterMap
	tmpLakeColor := color.RGBA{0, 0xFF, 0xFF, 0xFF}     // alpha=255 表示不透明
	tmpLakeColor2 := color.RGBA{0xd0, 0x46, 0x93, 0xFF} //#d04693    //purple
	tmpLakeColor3 := color.RGBA{0xa0, 0xd2, 0xeb, 0xFF} //#b0d2eb    // blue-gray
	for _, dot := range w.data {
		// 绘制积水 点周围绘制
		if dot.h > 0 {
			img.Set(int(dot.x)*zoom+zoom/2, int(dot.y)*zoom+zoom/2, tmpLakeColor) // self
			img.Set(int(dot.x)*zoom+zoom/2+1, int(dot.y)*zoom+zoom/2+1, tmpLakeColor)
			img.Set(int(dot.x)*zoom+zoom/2+1, int(dot.y)*zoom+zoom/2, tmpLakeColor)
			img.Set(int(dot.x)*zoom+zoom/2+1, int(dot.y)*zoom+zoom/2-1, tmpLakeColor)
			img.Set(int(dot.x)*zoom+zoom/2, int(dot.y)*zoom+zoom/2-1, tmpLakeColor)
			img.Set(int(dot.x)*zoom+zoom/2-1, int(dot.y)*zoom+zoom/2-1, tmpLakeColor)
			img.Set(int(dot.x)*zoom+zoom/2-1, int(dot.y)*zoom+zoom/2, tmpLakeColor)
			img.Set(int(dot.x)*zoom+zoom/2-1, int(dot.y)*zoom+zoom/2+1, tmpLakeColor)
			img.Set(int(dot.x)*zoom+zoom/2, int(dot.y)*zoom+zoom/2+1, tmpLakeColor)
		}
		if dot.q > 0 {
			img.Set(int(dot.x)*zoom+zoom/2+1, int(dot.y)*zoom+zoom/2, tmpLakeColor2)
			img.Set(int(dot.x)*zoom+zoom/2, int(dot.y)*zoom+zoom/2-1, tmpLakeColor2)
			img.Set(int(dot.x)*zoom+zoom/2-1, int(dot.y)*zoom+zoom/2, tmpLakeColor2)
			img.Set(int(dot.x)*zoom+zoom/2, int(dot.y)*zoom+zoom/2+1, tmpLakeColor2)
		}
	}

	// 绘制流动 在v2下相当于场
	for di, dot := range w.data {
		// 绘制当前点 如果是源头 则绘制白色
		if dot.xPower != 0.0 || dot.yPower != 0.0 {
			// 计算相对比例尺的高度
			tmpLevel := int(m.data[di]) + dot.h
			tmpLevel = int(float32(cslen*tmpLevel) / maxColor)
			// 防止越界
			if tmpLevel >= len(cs) {
				tmpLevel = len(cs) - 1
			}
			if tmpLevel < 0 {
				tmpLevel = 0
			}

			// 绘制流动方向 考虑缩放
			tmpColor := cs[tmpLevel]
			lineTo(img, int(dot.x)*zoom+zoom/2, int(dot.y)*zoom+zoom/2, int(dot.x)*zoom+zoom/2+int(float32(zoom)*dot.xPower), int(dot.y)*zoom+zoom/2+int(float32(zoom)*dot.yPower), color.RGBA{0, 0, 0xFF, 0xFF}, tmpColor, riverArrowScale)
		}
	}
	// 绘制droplets
	for _, drop := range drops {
		for _, dxi := range drop.hisway {
			img.Set(int(dxi%width)*zoom+zoom/2+1, int(dxi/width)*zoom+zoom/2+1, tmpLakeColor3)
			img.Set(int(dxi%width)*zoom+zoom/2+1, int(dxi/width)*zoom+zoom/2-1, tmpLakeColor3)
			img.Set(int(dxi%width)*zoom+zoom/2-1, int(dxi/width)*zoom+zoom/2-1, tmpLakeColor3)
			img.Set(int(dxi%width)*zoom+zoom/2-1, int(dxi/width)*zoom+zoom/2+1, tmpLakeColor3)
		}
	}
	// 绘制颜色模板
	for i := 0; i < len(cs); i++ {
		c := cs[len(cs)-i-1]
		for wi := 0; wi < 5; wi++ {
			img.Set(wi, i, c)
		}
	}
}

func DrawToConsole(m *Topomap) {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ~!@#$%^&*-=_+()[]{}<>\\/;:,.???????????????????????????????????????"
	for di, dd := range m.data {
		fmt.Printf("%c", str[dd])
		if di%m.width == (m.width - 1) {
			fmt.Printf("\n")
		}
	}
}

func DrawToHtml(w *WaterMap, m *Topomap) {
	// todo: use svg
	footerHtml := "<table>"
	for wi := 0; wi < w.width; wi++ {
		footerHtml += "<tr>"
		for hi := 0; hi < w.height; hi++ {
			idx := hi*w.height + wi
			tdot := &w.data[hi*w.height+wi]
			footerHtml += fmt.Sprintf(`<td title="xpower=%f ypower=%f h=%d" style="width:1px;height:1px;background:rgb(0,%d,0)">&nbsp;&nbsp;</td>`,
				tdot.xPower, tdot.yPower, m.data[idx], m.data[idx]*10)
		}
		footerHtml += "</tr>"
	}
	footerHtml += "</table>"

	drawer.SetHomeDrawHandler(func(rw http.ResponseWriter) {
		rw.Write([]byte(footerHtml))
	})
	log.Printf("draw to html ok")
}

func MakeRidge(ridgeLen, ridgeWide, width, height int) []Hill {
	ridgeHills := make([]Hill, ridgeLen)
	baseTowardX, baseTowardY := (rand.Int()%width-width/2)/20, (rand.Int()%height-height/2)/20
	for ri := 0; ri < int(ridgeLen); ri++ {
		r := &ridgeHills[ri]
		if ri == 0 {
			// 第一个
			r.x, r.y, r.r, r.h = (rand.Int() % width), (rand.Int() % height), (rand.Int()%(ridgeWide) + 1), (rand.Int()%(5) + 2)
		} else {
			// 其他
			r.x, r.y, r.r, r.h = ridgeHills[ri-1].x+(rand.Int()%ridgeWide)-ridgeWide/2+baseTowardX, ridgeHills[ri-1].y+(rand.Int()%ridgeWide)-ridgeWide/2+baseTowardY, (rand.Int()%(ridgeWide) + 1), (rand.Int()%(5) + 2)
		}
	}

	return ridgeHills
}

func ImgToFile(outputFilePath string, img *image.RGBA, format string) {
	picFile2, err := os.Create(outputFilePath)
	if err != nil {
		log.Printf("when create file %s error:%v", outputFilePath, err)
		return
	}
	defer picFile2.Close()
	if err := png.Encode(picFile2, img); err != nil {
		log.Println("png.Encode error:", err)
	}
}
