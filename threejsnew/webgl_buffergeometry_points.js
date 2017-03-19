
			if ( ! Detector.webgl ) Detector.addGetWebGLMessage();

			var container, stats;

			var camera, scene, renderer, controls, geometry;

			var points, positions, colors;
            // 可流程运行150W个particles 在chrome中150W占用内存3.8G 基本到极限
            var NUM_PARTICLES = 0;
            var ticker = 0;
            var color;
            var isInited = 0;
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
                clearOrbs();

                //var geometry = points.geometry;
                NUM_PARTICLES = list.length;
				positions = new Float32Array( NUM_PARTICLES * 3 );
				colors = new Float32Array( NUM_PARTICLES * 3 );

				var n = NUM_PARTICLES, n2 = n / 2; // particles spread in the cube

                //for ( var i = 0; i < positions.length; i += 3 ) {
                for (var i in list) {
                    var orb = list[i];

                    // positions
                    if (orb.st!=1) {
                        orb.x = orb.y = orb.z = 0;
                    }

                    var x = orb.x;
                    var y = orb.y;
                    var z = orb.z;

                    positions[ i*3 ]     = x*zoomBase;
                    positions[ i*3 + 1 ] = y*zoomBase;
                    positions[ i*3 + 2 ] = z*zoomBase;

                    // colors

                    var vx = ( x / n ) + 0.5;
                    var vy = ( y / n ) + 0.5;
                    var vz = ( z / n ) + 0.5;

                    color.setRGB( vx, vy, vz );

                    colors[ i*3 ]     = color.r;
                    colors[ i*3 + 1 ] = color.g;
                    colors[ i*3 + 2 ] = color.b;

                }

                geometry.addAttribute( 'position', new THREE.BufferAttribute( positions, 3 ) );
                geometry.addAttribute( 'color', new THREE.BufferAttribute( colors, 3 ) );

                geometry.computeBoundingSphere();

                var material;
                // 量大使用PointsMaterial渲染
                if (NUM_PARTICLES >= 20000) {
                    material = new THREE.PointsMaterial( { size: 200, vertexColors: THREE.VertexColors } );
                } else {
                    material = new THREE.PointsMaterial({ size: 200, map: sprite, blending: THREE.AdditiveBlending, depthTest: false, transparent : true });
                }
                //var programStroke = function ( context ) {
                //    context.lineWidth = 0.025;
                //    context.beginPath();
                //    context.arc( 0, 0, 0.5, 0, Math.PI * 2, true );
                //    context.stroke();
                //};
                //var material = new THREE.SpriteCanvasMaterial( { color: Math.random() * 0x808080 + 0x808080, program: programStroke } );


                points = new THREE.Points( geometry, material );
                scene.add( points );
                isInited = 1;
            }
            updateOrbs = function(list) {
                if (list.length != NUM_PARTICLES) {
                    return initOrbs(list);
                }
                var geometry = points.geometry;
                for (var i in list) {
                    var orb = list[i];

                    positions[ i*3 ]     = orb.x*zoomBase;
                    positions[ i*3 + 1 ] = orb.y*zoomBase;
                    positions[ i*3 + 2 ] = orb.z*zoomBase;
                }

                geometry.addAttribute( 'position', new THREE.BufferAttribute( positions, 3 ) );
                geometry.addAttribute( 'color', new THREE.BufferAttribute( colors, 3 ) );

                geometry.computeBoundingSphere();

            }


			//init();
			//animate();

			function init() {

				container = document.getElementById( 'container' );

                color = new THREE.Color();

				camera = new THREE.PerspectiveCamera( 27, window.innerWidth / window.innerHeight, 5, 350000 );
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
                sprite = new THREE.TextureLoader().load( "./textures/spark1.png" );				//

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
