# go语言并行模拟万有引力天体运行

服务端使用go利用多核并行计算，支持memcache/redis/file存储计算数据，前后端使用websocket通信，前端使用ThreeJS框架，利用WebGL显示3D效果。

效果图：

<img src="https://github.com/uxff/gravity_sim_go/raw/master/image/gravity_sim_20170430204800.jpg">
<img src="https://github.com/uxff/gravity_sim_go/raw/master/image/gravity_sim_20170430213314.jpg">
<img src="https://github.com/uxff/gravity_sim_go/raw/master/image/gravity_sim_20170430210511.jpg">


## 思路：

有两个程序,calc_server和websocket_server  
calc_server -> 计算数据保存到file/mc/redis  
浏览器查看 -> websocket_server ->读取file/mc/redis的数据  

查看需要浏览器支持WebGL,比如chrome/firefox。
ie各个版本/360急速/360安全浏览器无法查看。
chrome下支持100万粒子查看,并保持50+fps的流畅度,内存占用0.9-1.8G浮动。

## 使用方法：

```
#编译计算服务器
go build go_server/calc_server.go
#step 1.初始化数据
#这里初始化1000个，设置初始速度，初始质量，中心恒星质量
./calc_server -init-orbs 1000 -config-velo 0.05 -config-mass 100 -bigmass 18000
#step 2.执行演化10000次 可能会时间比较长 最后开新窗口执行下面
./calc_server -calc-times 10000
#编辑通信服务器
go build go_server/websocket_server.go
#step 3.运行通信服务器
./websocket_server -addr 192.168.12.150:8081
#step 4.使用chrome或firefox打开http://192.168.12.150:8081/index.html查看运行情况(需要浏览器支持WebGL和websocket)
#如果step 2步执行完毕，会自动保存数据到memcache，mc默认使用127.0.0.1:11211。
#继续执行step 2步，回到浏览器查看效果。

```
使用 calc_server -h 和 websocket_server -h 查看对应的服务端参数和用法。
## calc_server 参数说明：
```
  -bigmass float
    	config of big mass orb, like a blackhole in center, 0 means no bigger (default 15000)
        中心天体质量 0表示没有大质量天体 默认15000
  -bignum int
    	config of number of big mass orbs, generally center has 1 (default 1)
        中心大质量天体数量 通常为1个
  -bigstyle int
    	config of big mass orb distribute style: 0=center 1=outer edge 2=middle of one radius 3=random
        中心大质量天体分布方式 0=中央 1=外边缘 2=半径的中点 3=随机
  -calc-times int
    	how many times calc (default 100)
        计算次数
  -config-arrange int
    	init style of orbs arrangement 0=line,1=cube,2=disc,3=sphere (default 3)
        初始化时排列方式 0=线性(在线周围会有浮动) 1=立方体 2=盘状 3=球形
  -config-assemble int
    	init style of orbs aggregation 0=avg,1=ladder,2=variance,3=4th power (default 2)
        初始化聚集方式 0=平均分布 1=梯形分布 2=平方差 3=4次方差
  -config-cpu int
    	how many cpu u want use, 0=all
        使用多少cpu
  -config-mass float
    	init mass of orbs (default 10)
        初始化天体质量范围 默认10
  -config-velo float
    	init velo of orbs (default 0.005)
        初始化天体速度范围 默认0.005 不要太快，快了中心天体束缚不住
  -config-wide float
    	init wide of orbs (default 1000)
        初始化分布宽度范围 默认1000
  -domerge
    	merge from loadkey to savekey if true, replace if false
        合并loadkey中的天体列表到savekey中
  -init-orbs int
    	how many orbs init, do init when its value >1
        初始化天体数量 大于1将初始化 不传将使用loadkey的天体列表数据
  -loadkey string
    	key name to load, like key of memcache, or filename in save dir, use savekey if no given
        要加载的天体列表名，可以是存在memcache的key,可以是文件名 没指定则使用savekey指定的值
  -loadpath string
    	where to load, support mc/file/redis
        从哪里加载天体列表，支持memcache,file,redis. 没指定使用savepath指定的值
	like: file://./filecache/, use savepath if no given
    举例：文件方式当前目录下的filecache目录下 file://./filecache/ ;memcache方式 mc://127.0.0.1:11211 ;redis方式 redis://127.0.0.1:6179
  -moveexp string
    	move expression, like: x=-150&vy=+0.01&m=+20 only position,velo,mass
        使用表达式批量移动列表数据 只支持坐标，速度，质量 不支持数量,加速度,id
  -save-duration int
    	save to savepath per millisecond, 100 means 100ms (default 500)
        保存间隔 毫秒 1500=计算时每1.5秒保存一次 保存过于频繁影响计算速度，过慢影响浏览器查看效果
  -savekey string
    	key name to save, like key of memcache, or filename in save dir (default "thelist1")
        要保存的天体列表名
  -savepath string
    	where to save, support mc/file/redis
        在哪里保存天体列表数据，格式同loadpath
	like: file://./filecache/ (default "mc://127.0.0.1:11211")
  -showlist
    	show orb list and exit
        显示列表数据，并退出，不继续计算
```
