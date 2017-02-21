
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
            
            recvData = function(dataList) {
                //console.log('list=', dataList);
                if (dataList == undefined) {
                    console.warn('list is undefined');
                    return ;
                }
                if (!isInited) {
                    initOrbs(dataList);
                } else {
                    updateOrbs(dataList);
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

				var n = 1000, n2 = n / 2; // particles spread in the cube

                //for ( var i = 0; i < positions.length; i += 3 ) {
                for (var i in list) {
                    var orb = list[i];

                    // positions

                    var x = orb.x;
                    var y = orb.y;
                    var z = orb.z;

                    positions[ i*3 ]     = x;
                    positions[ i*3 + 1 ] = y;
                    positions[ i*3 + 2 ] = z;

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

                //var material = new THREE.PointsMaterial( { size: 20, vertexColors: THREE.VertexColors } );
                //var programStroke = function ( context ) {
                //    context.lineWidth = 0.025;
                //    context.beginPath();
                //    context.arc( 0, 0, 0.5, 0, Math.PI * 2, true );
                //    context.stroke();
                //};
				var sprite = new THREE.TextureLoader().load( "./textures/spark1.png" );
                var material = new THREE.PointsMaterial( { size: 20, map: sprite, blending: THREE.AdditiveBlending, depthTest: false, transparent : true } );
                //var material = new THREE.SpriteCanvasMaterial( { color: Math.random() * 0x808080 + 0x808080, program: programStroke } );


                points = new THREE.Points( geometry, material );
                scene.add( points );
                isInited = 1;
            }
            updateOrbs = function(list) {
                var geometry = points.geometry;
                for (var i in list) {
                    var orb = list[i];

                    positions[ i*3 ]     = orb.x;
                    positions[ i*3 + 1 ] = orb.y;
                    positions[ i*3 + 2 ] = orb.z;
                }

                geometry.addAttribute( 'position', new THREE.BufferAttribute( positions, 3 ) );
                geometry.addAttribute( 'color', new THREE.BufferAttribute( colors, 3 ) );

                geometry.computeBoundingSphere();

            }


			//init();
			//animate();

			function init() {

				container = document.getElementById( 'container' );

				//
                color = new THREE.Color();

				camera = new THREE.PerspectiveCamera( 27, window.innerWidth / window.innerHeight, 5, 35000 );
				camera.position.z = 2750;

				scene = new THREE.Scene();
				//scene.fog = new THREE.Fog( 0x050505, 2000, 3500 );
                controls = new THREE.OrbitControls(camera, container);
                controls.target = new THREE.Vector3(0, 0, 0);
                // 坐标系
                var axisHelper = new THREE.AxisHelper(1000); // 500 is size
                scene.add(axisHelper);

				//

				geometry = new THREE.BufferGeometry();

				//

				renderer = new THREE.WebGLRenderer( { antialias: false } );
				renderer.setClearColor( 0x0F0F0F );
				renderer.setPixelRatio( window.devicePixelRatio );
				renderer.setSize( window.innerWidth, window.innerHeight );

				container.appendChild( renderer.domElement );

				//

				stats = new Stats();
				container.appendChild( stats.dom );

				//

				window.addEventListener( 'resize', onWindowResize, false );

                // init websocket
                if (wsUri == undefined || wsUri.length==0) {
                    wsUri = $('#ws-addr').val();
                    if (window.document.domain != undefined) {
                        
                        wsUri = 'ws://'+window.document.domain+':8081'+'/orbs';
                        $('#ws-addr').val(wsUri);
                        MyWebsocket.wsUri = wsUri;
                    }
                }
                if (wsUri.length > 0) {
                    MyWebsocket.initWebsocket();
                }
                MyWebsocket.receiveCallback = recvData;

                $('#reConnect').on('click', function() {
                    wsUri = 'ws://'+$('#ws-addr').val()+'/orbs';
                    MyWebsocket.wsUri = wsUri;
                    //alert(wsUri);
                    MyWebsocket.initWebsocket();
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
                if (ticker%20==0) {
                    //updateDots();
                    MyWebsocket.doSend('k='+mcKey);
                }
                renderer.render( scene, camera );


			}

            if (window.addEventListener)
                window.addEventListener('load', init, false);
            else if (window.attachEvent)
                window.attachEvent('onload', init);
            else window.onload = init;
