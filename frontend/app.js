const API='/api/v1',UFA=[54.7351,55.9587],DEFAULT_START={lat:54.73348,lon:55.949477},DEFAULT_FINISH={lat:54.729533,lon:55.956415},state={start:{...DEFAULT_START},finish:{...DEFAULT_FINISH},picking:null,markers:{},route:null,mode:'walking'};
const $=s=>document.querySelector(s),$$=s=>[...document.querySelectorAll(s)],form=$('#routeForm'),message=$('#formMessage');
let mapAdapter;
new MutationObserver(()=>$('#mapHint').classList.toggle('hidden',!$('#routeSummary').classList.contains('hidden'))).observe($('#routeSummary'),{attributes:true,attributeFilter:['class']});
window.showPlannerTab=tab=>{const preferences=tab==='preferences';$$('[data-tab]').forEach(x=>x.classList.toggle('active',x.dataset.tab===tab));$('#preferencesPanel').classList.toggle('hidden',!preferences);form.classList.toggle('hidden',preferences);$('#results').classList.add('hidden')};

function leafletAdapter(){
  const map=L.map('map',{zoomControl:false}).setView(UFA,14);L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png',{maxZoom:19,attribution:'© OpenStreetMap'}).addTo(map);L.control.zoom({position:'bottomleft'}).addTo(map);
  return {provider:'osm',onClick:fn=>map.on('click',e=>fn({lat:e.latlng.lat,lon:e.latlng.lng})),marker(kind,p,label){if(state.markers[kind])state.markers[kind].remove();state.markers[kind]=L.marker([p.lat,p.lon]).addTo(map).bindTooltip(label,{permanent:true,direction:'top',offset:[0,-12]})},setView:p=>map.setView([p.lat,p.lon],15),fit:(a,b)=>map.fitBounds([[a.lat,a.lon],[b.lat,b.lon]],{padding:[60,60]}),draw(feature,points){if(state.route)state.route.remove();state.route=L.geoJSON(feature,{style:{color:'#3c2c29',weight:6,opacity:.9,lineCap:'round'}}).addTo(map);points.filter(p=>p.type==='place').forEach(p=>L.circleMarker([p.location.lat,p.location.lon],{radius:12,color:'#fff',weight:4,fillColor:'#ff625f',fillOpacity:1}).addTo(state.route).bindPopup(`<strong>${esc(p.place_name)}</strong><br>${esc(p.category)}<br>${p.duration_min} мин.`));map.fitBounds(state.route.getBounds(),{padding:[45,45]})}};
}

async function yandexAdapter(key){
  await new Promise((resolve,reject)=>{const script=document.createElement('script');script.src=`https://api-maps.yandex.ru/2.1/?apikey=${encodeURIComponent(key)}&lang=ru_RU`;script.onload=()=>ymaps.ready(resolve);script.onerror=reject;document.head.appendChild(script)});
  const map=new ymaps.Map('map',{center:UFA,zoom:14,controls:['zoomControl','geolocationControl']});
  return {provider:'yandex',onClick:fn=>map.events.add('click',e=>{const [lat,lon]=e.get('coords');fn({lat,lon})}),marker(kind,p,label){if(state.markers[kind])map.geoObjects.remove(state.markers[kind]);state.markers[kind]=new ymaps.Placemark([p.lat,p.lon],{iconCaption:label},{preset:kind==='start'?'islands#blueCircleDotIcon':'islands#redCircleDotIcon'});map.geoObjects.add(state.markers[kind])},setView:p=>map.setCenter([p.lat,p.lon],15),fit:(a,b)=>map.setBounds([[Math.min(a.lat,b.lat),Math.min(a.lon,b.lon)],[Math.max(a.lat,b.lat),Math.max(a.lon,b.lon)]],{checkZoomRange:true,zoomMargin:45}),draw(feature,points){if(state.route)map.geoObjects.remove(state.route);const modes={walking:'pedestrian',metro:'masstransit',bus:'masstransit',car:'auto'},referencePoints=points.map(p=>[p.location.lat,p.location.lon]);if(ymaps.multiRouter&&referencePoints.length>1){state.route=new ymaps.multiRouter.MultiRoute({referencePoints,params:{routingMode:modes[state.mode]||'pedestrian',results:1}},{boundsAutoApply:true,routeActiveStrokeColor:'#3c2c29',routeActiveStrokeWidth:6});map.geoObjects.add(state.route);return}state.route=new ymaps.GeoObject({geometry:{type:'LineString',coordinates:feature.geometry.coordinates.map(([lon,lat])=>[lat,lon])}},{strokeColor:'#3c2c29',strokeWidth:6,strokeOpacity:.9});map.geoObjects.add(state.route);const c=feature.geometry.coordinates;map.setBounds([[Math.min(...c.map(x=>x[1])),Math.min(...c.map(x=>x[0]))],[Math.max(...c.map(x=>x[1])),Math.max(...c.map(x=>x[0]))]],{checkZoomRange:true,zoomMargin:45})}};
}

async function initMap(){
  const key=window.APP_CONFIG?.YANDEX_MAPS_API_KEY?.trim();
  if(key){try{mapAdapter=await yandexAdapter(key);$('#mapHint').textContent='Яндекс Карты · выберите старт и финиш'}catch(e){console.warn('Yandex Maps unavailable, falling back to OSM',e);mapAdapter=leafletAdapter()}}else mapAdapter=leafletAdapter();
  mapAdapter.onClick(point=>{const kind=state.picking||'finish';setMarker(kind,point,kind==='start'?'Старт':'Финиш');reverseGeocode(point).then(a=>$(`#${kind}Input`).value=a).catch(()=>{})});
  try{const [a,b]=await Promise.all([geocode($('#startInput').value),geocode($('#finishInput').value)]);setMarker('start',a,'Старт');setMarker('finish',b,'Финиш');mapAdapter.fit(a,b)}catch(e){console.warn(e)}
}

function minutesBetween(a,b){const x=a.split(':').map(Number),y=b.split(':').map(Number),d=y[0]*60+y[1]-x[0]*60-x[1];return d>0?d:d+1440}
function setMarker(kind,p,label){state[kind]={lat:p.lat,lon:p.lon};mapAdapter.marker(kind,p,label);state.picking=kind==='start'?'finish':null;$('#mapHint').textContent=state.picking?'Теперь укажите финиш':'Точки выбраны — нажмите Go!'}
$$('[data-pick]').forEach(b=>b.addEventListener('click',()=>{state.picking=b.dataset.pick;$('#mapHint').textContent=state.picking==='start'?'Укажите старт на карте':'Укажите финиш на карте'}));

async function geocode(q){
  if(!q||!q.trim())return DEFAULT_START;
  if(mapAdapter?.provider==='yandex'&&window.ymaps?.geocode){
    try{const r=await ymaps.geocode(q,{results:1}),x=r.geoObjects.get(0);if(x){const [lat,lon]=x.geometry.getCoordinates();return{lat,lon}}}catch(e){console.warn('Yandex geocode error',e)}
  }
  try{
    const ctrl=new AbortController(),t=setTimeout(()=>ctrl.abort(),3000);
    const r=await fetch(`https://nominatim.openstreetmap.org/search?format=jsonv2&limit=1&countrycodes=ru&q=${encodeURIComponent(q)}`,{headers:{Accept:'application/json'},signal:ctrl.signal});
    clearTimeout(t);
    if(r.ok){const data=await r.json();if(data?.[0])return{lat:Number(data[0].lat),lon:Number(data[0].lon)}}
  }catch(e){console.warn('Nominatim error',e)}
  if(q.includes('Достоевского'))return DEFAULT_START;
  if(q.includes('Цюрупы'))return DEFAULT_FINISH;
  return{lat:UFA[0],lon:UFA[1]};
}
async function reverseGeocode(p){
  if(mapAdapter?.provider==='yandex'&&window.ymaps?.geocode){
    try{const r=await ymaps.geocode([p.lat,p.lon]),x=r.geoObjects.get(0);if(x?.getAddressLine())return x.getAddressLine()}catch(e){console.warn('Yandex reverse error',e)}
  }
  try{
    const ctrl=new AbortController(),t=setTimeout(()=>ctrl.abort(),3000);
    const r=await fetch(`https://nominatim.openstreetmap.org/reverse?format=jsonv2&lat=${p.lat}&lon=${p.lon}`,{signal:ctrl.signal});
    clearTimeout(t);
    if(r.ok){const d=await r.json();if(d?.display_name)return d.display_name.split(',').slice(0,3).join(', ')}
  }catch(e){console.warn('Nominatim reverse error',e)}
  return `${p.lat.toFixed(5)}, ${p.lon.toFixed(5)}`;
}
async function api(path,options={}){
  try{
    const r=await fetch(`${API}${path}`,{...options,headers:{'Content-Type':'application/json',...(options.headers||{})}}),d=await r.json().catch(()=>({}));
    if(!r.ok)throw Error(d.details||d.error||`Ошибка API ${r.status}`);
    return d;
  }catch(err){
    if(err.name==='TypeError'||err.message?.includes('NetworkError')){
      throw Error('Ошибка соединения с сервером. Пожалуйста, повторите запрос.');
    }
    throw err;
  }
}
function interests(){return Object.fromEntries($$('#interestPicker input').map(input=>[input.dataset.interest,Number(input.value)/10]))}
function preferenceValues(){return Object.fromEntries($$('#interestPicker input').map(input=>[input.dataset.interest,Number(input.value)]))}
function anonymousEmail(){let email=localStorage.getItem('routeGuestEmail');if(!email){const token=crypto.randomUUID?.()||`${Date.now()}-${Math.random().toString(16).slice(2)}`;email=`guest-${token}@go.local`;localStorage.setItem('routeGuestEmail',email)}return email}
async function ensureAnonymousUser(forceUpdate=false){const storedID=localStorage.getItem('routeUserId');if(storedID&&!forceUpdate){try{return await api(`/users/${storedID}`)}catch(error){localStorage.removeItem('routeUserId')}}const profile=await api('/users/',{method:'POST',body:JSON.stringify({name:'Путешественник',email:anonymousEmail(),interests:interests()})});localStorage.setItem('routeUserId',profile.id);return profile}
function loadPreferences(){let saved;try{saved=JSON.parse(localStorage.getItem('routeInterests')||'null')}catch(error){saved=null}$$('#interestPicker input').forEach(input=>{if(Array.isArray(saved))input.value=saved.includes(input.dataset.interest)?10:1;else if(saved&&Number.isFinite(Number(saved[input.dataset.interest])))input.value=Math.max(0,Math.min(10,Number(saved[input.dataset.interest])));input.closest('label').querySelector('output').value=input.value})}
async function ensurePoints(){const jobs=[];if(!state.start)jobs.push(geocode($('#startInput').value).then(p=>setMarker('start',p,'Старт')));if(!state.finish)jobs.push(geocode($('#finishInput').value).then(p=>setMarker('finish',p,'Финиш')));await Promise.all(jobs)}
function esc(v){const e=document.createElement('div');e.textContent=v??'';return e.innerHTML}
function render(route){mapAdapter.draw(route.geojson,route.waypoints);$('#summaryTime').textContent=`${route.total_duration_min} мин`;$('#summaryDistance').textContent=`${(route.total_distance_meters/1000).toFixed(1)} км`;$('#routeSummary').classList.remove('hidden');$('#resultTitle').textContent=`Маршрут на ${route.total_duration_min} минут`;$('#metrics').innerHTML=`<div><strong>${route.travel_duration_min??'—'} мин</strong><small>дорога</small></div><div><strong>${route.visit_duration_min??'—'} мин</strong><small>места</small></div><div><strong>${route.arrival_buffer_min??0} мин</strong><small>запас</small></div><div><strong>≈ ${route.estimated_cost_rub??0} ₽</strong><small>стоимость</small></div>`;const reasons=Array.isArray(route.match_reasons)?route.match_reasons:[route.match_reasons].filter(Boolean);$('#reasons').innerHTML=reasons.map(x=>`<span>${esc(x)}</span>`).join('');$('#waypoints').innerHTML=route.waypoints.map((p,i)=>{const rawName=String(p.place_name||''),safeName=/^[\d\s/\\.,№#-]+$/.test(rawName)?(p.category||'Интересное место'):rawName,n=p.type==='start'?'Начало маршрута':p.type==='finish'?'Финиш':safeName,d=p.type==='place'?`${p.category} · ${p.duration_min} мин · ${(p.distance_from_prev_meters||0).toFixed(0)} м`:`${p.location.lat.toFixed(4)}, ${p.location.lon.toFixed(4)}`;return`<li data-order="${i+1}"><strong>${esc(n)}</strong><span>${esc(d)}</span></li>`}).join('');form.classList.add('hidden');$('#results').classList.remove('hidden')}
form.addEventListener('submit',async e=>{e.preventDefault();const b=$('#goButton');b.disabled=true;message.textContent='Ищем места и проверяем дедлайн…';try{await ensurePoints();const profile=await ensureAnonymousUser();const route=await api('/routes/build',{method:'POST',body:JSON.stringify({user_ids:[profile.id],budget_minutes:minutesBetween($('#timeFrom').value,$('#timeTo').value),budget_rub:Number($('#budgetInput').value)||0,arrival_buffer_min:Number($('#bufferInput').value)||0,transport_mode:state.mode,start:state.start,finish:state.finish})});render(route)}catch(err){console.error(err);message.textContent=err.message.includes('no places')?'Места не найдены. Добавьте ключ 2GIS или OSM-данные.':err.message}finally{b.disabled=false}});
$('#editButton').addEventListener('click',()=>{$('#results').classList.add('hidden');form.classList.remove('hidden')});$('#togglePlanner').addEventListener('click',()=>$('#planner').classList.toggle('collapsed'));
$$('#interestPicker input').forEach(input=>input.addEventListener('input',()=>{input.closest('label').querySelector('output').value=input.value}));$$('#pacePicker button').forEach(b=>b.addEventListener('click',()=>{$$('#pacePicker button').forEach(x=>x.classList.remove('active'));b.classList.add('active')}));$$('#transportPicker button').forEach(b=>b.addEventListener('click',()=>{$$('#transportPicker button').forEach(x=>x.classList.remove('active'));b.classList.add('active');state.mode=b.dataset.mode}));
$('#savePreferences').addEventListener('click',async()=>{const status=$('#preferencesStatus');localStorage.setItem('routeInterests',JSON.stringify(preferenceValues()));status.textContent='Сохраняем…';try{await ensureAnonymousUser(true);status.textContent='Готово — предпочтения сохранены'}catch(error){console.error(error);status.textContent='Сохранено на устройстве. Сервер сейчас недоступен.'}});
$('#locateButton').addEventListener('click',()=>navigator.geolocation?.getCurrentPosition(p=>{const x={lat:p.coords.latitude,lon:p.coords.longitude};setMarker('start',x,'Вы здесь');mapAdapter.setView(x)},()=>$('#mapHint').textContent='Не удалось получить геопозицию'));
loadPreferences();initMap();
