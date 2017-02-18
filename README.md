# go语言并行模拟万有引力天体运行

服务端使用go利用多核并行计算，使用memcache存储计算数据，前后端使用websocket通信，前端使用ThreeJS框架，利用WebGL显示3D效果。

效果图：

<img src="http://wx3.sinaimg.cn/mw690/7a9cebc0ly1fcuzt0addej21hc0t8agc.jpg">
<img src="http://wx2.sinaimg.cn/mw690/7a9cebc0ly1fcuzt0ocjnj21hc0t8ach.jpg">

使用方法：

```
#编译计算服务器
go build go_server/calc_server.go
#s1.初始化数据
#这里初始化1000个，设置初始速度，初始质量，中心恒星质量
./calc_server -init-orbs 1000 -config-velo 0.05 -config-mass 100 -eternal 18000
#s2.执行演化10000次 可能会时间比较长 最后开新窗口执行下面
./calc_server -calc-times 10000
#编辑通信服务器
go build go_server/websocket_server.go
#s3.运行通信服务器
./websocket_server
#s4.使用chrome或firefox打开index.html查看运行情况 需要浏览器支持WebGL和websocket
#如果s2步执行完毕，会自动保存数据到memcache，mc默认使用127.0.0.1:11211。
#继续执行s2步，回到浏览器查看效果。

```
使用 calc_server -h 和 websocket_server -h 查看对应的服务端参数和用法。

