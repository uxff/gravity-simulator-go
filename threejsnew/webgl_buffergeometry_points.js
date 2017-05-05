
            if ( ! Detector.webgl ) Detector.addGetWebGLMessage();

            var container, stats;

            var camera, scene, renderer, controls, geometry;

            var points, positions, colors, sizes;
            // 可流程运行150W个particles 在chrome中150W占用内存3.8G 基本到极限
            var NUM_PARTICLES = 0;
            var ticker = 0;
            var color;
            var isInited = 0, initTimes = 0;
            var recvData, clearOrbs, initOrbs, updateOrbs;
            var zoomBase = 1.0, zoomStep = Math.sqrt(2.0);
            var sprite;
            
            recvData = function(dataList) {
                //console.log('list=', dataList);
                if (dataList == undefined) {
                    console.warn('list is undefined');
                    return ;
                }
                if (dataList.data.list != undefined) {
                    
                    if (!isInited) {
                        initOrbs(dataList.data.list);
                    } else {
                        updateOrbs(dataList.data.list);
                    }
                } else {
                    //console.log("unknown cmd:", dataList);
                }
            }
            clearOrbs = function() {
                //var geometry = points.geometry;
                //geometry.removeAttribute( 'position');
                //geometry.removeAttribute( 'color');
            }
            initOrbs = function(list) {
                ++initTimes;
                clearOrbs();

                // 使用精简数据格式{x,y,z,m}传输，提高网络使用率，降低chrome内存使用，使chrome支持100W粒子
                //var geometry = points.geometry;
                NUM_PARTICLES = list.length;
                positions = new Float32Array( NUM_PARTICLES * 3 );
                colors = new Float32Array( NUM_PARTICLES * 3 );
                sizes = new Float32Array( NUM_PARTICLES );
                /* custom color
                */
                var uniforms = {
                    color:     { value: new THREE.Color( 0xffffff ) },
                    texture:   { value: new THREE.TextureLoader().load( "textures/spark1.png" ) }
                };

                var shaderMaterial = new THREE.ShaderMaterial( {
                    uniforms:       uniforms,
                    vertexShader:   document.getElementById( 'vertexshader' ).textContent,
                    fragmentShader: document.getElementById( 'fragmentshader' ).textContent,

                    blending:       THREE.AdditiveBlending,
                    depthTest:      false,
                    transparent:    true
                });

                //for ( var i = 0; i < positions.length; i += 3 ) {
                for (var i in list) {
                    var orb = list[i];

                    // positions
                    //if (orb.st!=1) {
                    //    orb.x = orb.y = orb.z = 0;
                    //}

                    positions[ i*3 + 0 ] = orb[0]*zoomBase;
                    positions[ i*3 + 1 ] = orb[1]*zoomBase;
                    positions[ i*3 + 2 ] = orb[2]*zoomBase;

                    /* 经过测试 50W个orb 绘制显示fps在[15-45]范围内，基本良好 */
                    color.setHSL( orb[3]*11.5, 1.0, 0.5 );//color.setHSL( Math.random(), 1.0, 0.5 );//color.setHSL( orb.id / 2147483647, 1.0, 0.5 );//
                    colors[ i*3 + 0 ] = color.r;
                    colors[ i*3 + 1 ] = color.g;
                    colors[ i*3 + 2 ] = color.b;
                    sizes[ i ] = Math.sqrt(Math.sqrt(orb[3])) * 100;//sizes[ i ] = 100;//
                }

                geometry.addAttribute( 'position', new THREE.BufferAttribute( positions, 3 ) );
                geometry.addAttribute( 'customColor', new THREE.BufferAttribute( colors, 3 ) );
                geometry.addAttribute( 'size', new THREE.BufferAttribute( sizes, 1 ) );

                geometry.computeBoundingSphere();

                // 使用自带方块操作 数量到100W时，显示良好，fps大于50，内存0.9-1.3G
                //var material = new THREE.PointsMaterial( { size: 200, vertexColors: THREE.VertexColors } );
                // 使用图片 spark1.png 风格 数量100W时，显示良好，fqs大于50，内存1.1-1.8G
                //var material = new THREE.PointsMaterial({ size: 200, map: sprite, blending: THREE.AdditiveBlending, depthTest: false, transparent : true });
                //points = new THREE.Points( geometry, material );
                /* 使用 customColor 风格 数量100W时，显示良好，fqs大于50，内存0.8-1.3G */
                points = new THREE.Points( geometry, shaderMaterial );

                scene.add( points );
                isInited = 1;
            }
            updateOrbs = function(list) {
                if (list.length != NUM_PARTICLES) {
                    MyWebsocket.writeToScreen("RE INIT WHEN UPDATE, times:"+initTimes);
                    return initOrbs(list);
                }
                //MyWebsocket.writeToScreen("RE INIT WHEN UPDATE");
                var geometry = points.geometry;
                for (var i in list) {
                    var orb = list[i];

                    positions[ i*3 + 0 ] = orb[0]*zoomBase;
                    positions[ i*3 + 1 ] = orb[1]*zoomBase;
                    positions[ i*3 + 2 ] = orb[2]*zoomBase;
                    //sizes[ i ] = Math.sqrt(Math.sqrt(orb[3])) * 100;//sizes[ i ] = 100;//
                }

                //geometry.addAttribute( 'position', new THREE.BufferAttribute( positions, 3 ) );
                //geometry.addAttribute( 'color', new THREE.BufferAttribute( colors, 3 ) );
                geometry.attributes.position.needsUpdate = true;
                geometry.attributes.customColor.needsUpdate = true;
                geometry.attributes.size.needsUpdate = true;

                geometry.computeBoundingSphere();

            }

            //init();
            //animate();

            function init() {

                container = document.getElementById( 'container' );

                color = new THREE.Color();

                camera = new THREE.PerspectiveCamera( 27, window.innerWidth / window.innerHeight, 10, 5000000 );
                camera.position.x = 1200;
                camera.position.y = 2400;
                camera.position.z = 1000;

                scene = new THREE.Scene();
                //scene.fog = new THREE.Fog( 0x050505, 2000, 3500 );
                controls = new THREE.OrbitControls(camera, container);
                controls.target = new THREE.Vector3(0, 0, 0);
                // 坐标系
                var axisHelper = new THREE.AxisHelper(1000); // 500 is size
                scene.add(axisHelper);

                //
                sprite = new THREE.TextureLoader().load( "./textures/spark1.png" );                //

                geometry = new THREE.BufferGeometry();

                //
                renderer = new THREE.WebGLRenderer( { antialias: false } );
                renderer.setClearColor( 0x0F0F0F );
                renderer.setPixelRatio( window.devicePixelRatio );
                renderer.setSize( window.innerWidth, window.innerHeight );

                container.appendChild( renderer.domElement );

                //
                stats = new Stats();
                stats.domElement.style.position = 'absolute';
                stats.domElement.style.top = (window.innerHeight - 50 )+'px';
                stats.domElement.style.zIndex = 10;
                container.appendChild( stats.dom );

                //
                window.addEventListener( 'resize', onWindowResize, false );

                // init websocket
                if (wsUri == undefined || wsUri.length==0) {
                    wsUri = $('#ws-addr').val();
                    if (window.document.domain != undefined) {
                        
                        wsUri = 'ws://'+window.document.location.host+'/orbs';
                        $('#ws-addr').val(wsUri);
                        MyWebsocket.wsUri = wsUri;
                    }
                }
                if (wsUri.length > 0) {
                    MyWebsocket.initWebsocket();
                }
                MyWebsocket.receiveCallback = recvData;

                $('#ws-addr').val(wsUri);
                $('#zoom_up').on('click', function() {
                    zoomBase = zoomBase*zoomStep;
                    $('#zoom').val(zoomBase);
                });
                $('#zoom_down').on('click', function() {
                    zoomBase = zoomBase/zoomStep;
                    $('#zoom').val(zoomBase);
                });
                $('#reConnect').on('click', function() {
                    wsUri = $('#ws-addr').val();
                    MyWebsocket.wsUri = wsUri;
                    //alert(wsUri);
                    MyWebsocket.initWebsocket();
                });
                $('#btnSend').on('click', function() {
                    sendVal = $('#send-val').val();
                });
                animate();
            }

            function onWindowResize() {

                camera.aspect = window.innerWidth / window.innerHeight;
                camera.updateProjectionMatrix();

                renderer.setSize( window.innerWidth, window.innerHeight );
            }

            //

            function animate() {

                requestAnimationFrame( animate );

                render();
                stats.update();

            }

            function render() {

                //var time = Date.now() * 0.001;
                ++ticker;

                //points.rotation.x = time * 0.25;
                //points.rotation.y = time * 0.5;
                if (ticker%15==0) {
                    //updateDots();
                    MyWebsocket.doSend(sendVal);
                }
                renderer.render( scene, camera );
            }

            if (window.addEventListener)
                window.addEventListener('load', init, false);
            else if (window.attachEvent)
                window.attachEvent('onload', init);
            else window.onload = init;
