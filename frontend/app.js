const API='/api/v1',UFA=[54.7351,55.9587],DEFAULT_START={lat:54.73348,lon:55.949477},DEFAULT_FINISH={lat:54.729533,lon:55.956415};
const TWOGIS_KEY='02340044-2020-4a6c-9d81-985d61c40f93';
const state={start:{...DEFAULT_START},finish:{...DEFAULT_FINISH},startAddress:'',finishAddress:'',picking:null,markers:{},route:null,mode:'walking'};
const $=s=>document.querySelector(s),$$=s=>[...document.querySelectorAll(s)],form=$('#routeForm'),message=$('#formMessage');
let mapAdapter;

new MutationObserver(()=>$('#mapHint').classList.toggle('hidden',!$('#routeSummary').classList.contains('hidden'))).observe($('#routeSummary'),{attributes:true,attributeFilter:['class']});
window.showPlannerTab=tab=>{const preferences=tab==='preferences';$$('[data-tab]').forEach(x=>x.classList.toggle('active',x.dataset.tab===tab));$('#preferencesPanel').classList.toggle('hidden',!preferences);form.classList.toggle('hidden',preferences);$('#results').classList.add('hidden')};

function leafletAdapter(){
  const map=L.map('map',{zoomControl:false}).setView(UFA,14);
  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png',{maxZoom:19,attribution:'© OpenStreetMap'}).addTo(map);
  L.control.zoom({position:'bottomleft'}).addTo(map);
  return {
    provider:'osm',
    onClick:fn=>map.on('click',e=>fn({lat:e.latlng.lat,lon:e.latlng.lng})),
    marker(kind,p,label){
      if(state.markers[kind])state.markers[kind].remove();
      state.markers[kind]=L.marker([p.lat,p.lon]).addTo(map).bindTooltip(label,{permanent:true,direction:'top',offset:[0,-12]});
    },
    setView:p=>map.setView([p.lat,p.lon],15),
    fit:(a,b)=>{
      if(!a||!b)return;
      if(Math.abs(a.lat-b.lat)<0.0001&&Math.abs(a.lon-b.lon)<0.0001){
        map.setView([a.lat,a.lon],15);
      }else{
        map.fitBounds([[a.lat,a.lon],[b.lat,b.lon]],{padding:[60,60]});
      }
    },
    clearRoute(){
      if(state.route){
        state.route.remove();
        state.route=null;
      }
    },
    draw(feature,points){
      this.clearRoute();
      state.route=L.geoJSON(feature,{style:{color:'#3c2c29',weight:6,opacity:.9,lineCap:'round'}}).addTo(map);
      points.filter(p=>p.type==='place').forEach(p=>L.circleMarker([p.location.lat,p.location.lon],{radius:12,color:'#fff',weight:4,fillColor:'#ff625f',fillOpacity:1}).addTo(state.route).bindPopup(`<strong>${esc(p.place_name)}</strong><br>${esc(p.category)}<br>${p.duration_min} мин.`));
      map.fitBounds(state.route.getBounds(),{padding:[45,45]});
    }
  };
}

async function yandexAdapter(key){
  await new Promise((resolve,reject)=>{
    const script=document.createElement('script');
    script.src=`https://api-maps.yandex.ru/2.1/?apikey=${encodeURIComponent(key)}&lang=ru_RU`;
    script.onload=()=>ymaps.ready(resolve);
    script.onerror=reject;
    document.head.appendChild(script);
  });
  const map=new ymaps.Map('map',{center:UFA,zoom:14,controls:['zoomControl','geolocationControl']});
  return {
    provider:'yandex',
    onClick:fn=>map.events.add('click',e=>{const [lat,lon]=e.get('coords');fn({lat,lon})}),
    marker(kind,p,label){
      if(state.markers[kind])map.geoObjects.remove(state.markers[kind]);
      state.markers[kind]=new ymaps.Placemark([p.lat,p.lon],{iconCaption:label},{preset:kind==='start'?'islands#blueCircleDotIcon':'islands#redCircleDotIcon'});
      map.geoObjects.add(state.markers[kind]);
    },
    setView:p=>map.setCenter([p.lat,p.lon],15),
    fit:(a,b)=>{
      if(!a||!b)return;
      if(Math.abs(a.lat-b.lat)<0.0001&&Math.abs(a.lon-b.lon)<0.0001){
        map.setCenter([a.lat,a.lon],15);
      }else{
        map.setBounds([[Math.min(a.lat,b.lat),Math.min(a.lon,b.lon)],[Math.max(a.lat,b.lat),Math.max(a.lon,b.lon)]],{checkZoomRange:true,zoomMargin:45});
      }
    },
    clearRoute(){
      if(state.route){
        map.geoObjects.remove(state.route);
        state.route=null;
      }
    },
    draw(feature,points){
      this.clearRoute();
      const modes={walking:'pedestrian',metro:'masstransit',bus:'masstransit',car:'auto'},referencePoints=points.map(p=>[p.location.lat,p.location.lon]);
      if(ymaps.multiRouter&&referencePoints.length>1){
        state.route=new ymaps.multiRouter.MultiRoute({referencePoints,params:{routingMode:modes[state.mode]||'pedestrian',results:1}},{boundsAutoApply:false,routeActiveStrokeColor:'#3c2c29',routeActiveStrokeWidth:6});
        map.geoObjects.add(state.route);
        map.setBounds([[Math.min(...referencePoints.map(p=>p[0])),Math.min(...referencePoints.map(p=>p[1]))],[Math.max(...referencePoints.map(p=>p[0])),Math.max(...referencePoints.map(p=>p[1]))]],{checkZoomRange:true,zoomMargin:45});
        return;
      }
      state.route=new ymaps.GeoObject({geometry:{type:'LineString',coordinates:feature.geometry.coordinates.map(([lon,lat])=>[lat,lon])}},{strokeColor:'#3c2c29',strokeWidth:6,strokeOpacity:.9});
      map.geoObjects.add(state.route);
      const c=feature.geometry.coordinates;
      map.setBounds([[Math.min(...c.map(x=>x[1])),Math.min(...c.map(x=>x[0]))],[Math.max(...c.map(x=>x[1])),Math.max(...c.map(x=>x[0]))]],{checkZoomRange:true,zoomMargin:45});
    }
  };
}

async function initMap(){
  const key=window.APP_CONFIG?.YANDEX_MAPS_API_KEY?.trim();
  if(key){
    try{
      mapAdapter=await yandexAdapter(key);
      $('#mapHint').textContent='Яндекс Карты · выберите старт и финиш';
    }catch(e){
      console.warn('Yandex Maps unavailable, falling back to OSM',e);
      mapAdapter=leafletAdapter();
    }
  }else{
    mapAdapter=leafletAdapter();
  }

  mapAdapter.onClick(point=>{
    const kind=state.picking||'finish';
    setMarker(kind,point,kind==='start'?'Старт':'Финиш');
    reverseGeocode(point).then(a=>{
      $(`#${kind}Input`).value=a;
      state[`${kind}Address`]=a;
    }).catch(()=>{});
  });

  const startVal=$('#startInput').value.trim();
  const finishVal=$('#finishInput').value.trim();
  try{
    const [a,b]=await Promise.all([geocode(startVal),geocode(finishVal)]);
    setMarker('start',a,'Старт');
    state.startAddress=startVal;
    setMarker('finish',b,'Финиш');
    state.finishAddress=finishVal;
    mapAdapter.fit(a,b);
  }catch(e){
    console.warn('Initial geocoding warning',e);
  }
}

function minutesBetween(a,b){
  const x=a.split(':').map(Number),y=b.split(':').map(Number),d=y[0]*60+y[1]-x[0]*60-x[1];
  return d>0?d:d+1440;
}

function setMarker(kind,p,label){
  state[kind]={lat:p.lat,lon:p.lon};
  mapAdapter.marker(kind,p,label);
  state.picking=kind==='start'?'finish':null;
  $('#mapHint').textContent=state.picking?'Теперь укажите финиш':'Точки выбраны — нажмите Go!';
}

$$('[data-pick]').forEach(b=>b.addEventListener('click',()=>{
  state.picking=b.dataset.pick;
  $('#mapHint').textContent=state.picking==='start'?'Укажите старт на карте':'Укажите финиш на карте';
}));

async function handleAddressInput(kind){
  const input=$(`#${kind}Input`);
  const val=input.value.trim();
  if(!val)return;
  message.textContent='Ищем адрес…';
  try{
    const p=await geocode(val);
    setMarker(kind,p,kind==='start'?'Старт':'Финиш');
    state[`${kind}Address`]=val;

    // If user changed start city and finish is still old Ufa default, auto-sync finish to start location
    if(kind==='start'&&$('#finishInput').value.includes('Цюрупы')&&!val.includes('Уфа')){
      $('#finishInput').value=val;
      setMarker('finish',p,'Финиш');
      state.finishAddress=val;
    }

    if(state.start&&state.finish){
      mapAdapter.fit(state.start,state.finish);
    }else{
      mapAdapter.setView(p);
    }
    message.textContent='Адрес найден! Нажмите Go! для построения маршрута.';
  }catch(e){
    console.warn(e);
    message.textContent='Адрес не найден. Уточните название или укажите точку на карте.';
  }
}

$('#startInput').addEventListener('change',()=>handleAddressInput('start'));
$('#finishInput').addEventListener('change',()=>handleAddressInput('finish'));

async function geocode(q){
  if(!q||!q.trim())return DEFAULT_START;
  const query=q.trim();

  // 1. 2GIS Geocoder API
  try{
    const ctrl=new AbortController(),t=setTimeout(()=>ctrl.abort(),4000);
    const r=await fetch(`https://catalog.api.2gis.com/3.0/items/geocode?q=${encodeURIComponent(query)}&key=${TWOGIS_KEY}&fields=items.point`,{signal:ctrl.signal});
    clearTimeout(t);
    if(r.ok){
      const data=await r.json();
      const pt=data?.result?.items?.[0]?.point;
      if(pt?.lat&&pt?.lon)return{lat:Number(pt.lat),lon:Number(pt.lon)};
    }
  }catch(e){
    console.warn('2GIS geocode error',e);
  }

  // 2. Yandex Maps Geocoder
  if(mapAdapter?.provider==='yandex'&&window.ymaps?.geocode){
    try{
      const r=await ymaps.geocode(query,{results:1});
      const x=r.geoObjects.get(0);
      if(x){
        const [lat,lon]=x.geometry.getCoordinates();
        return{lat,lon};
      }
    }catch(e){
      console.warn('Yandex geocode error',e);
    }
  }

  // 3. Fallbacks for standard demo addresses only
  if(query.includes('Достоевского'))return DEFAULT_START;
  if(query.includes('Цюрупы'))return DEFAULT_FINISH;

  throw new Error(`Адрес не найден: ${query}`);
}

async function reverseGeocode(p){
  // 1. 2GIS Reverse Geocoder
  try{
    const ctrl=new AbortController(),t=setTimeout(()=>ctrl.abort(),4000);
    const r=await fetch(`https://catalog.api.2gis.com/3.0/items/geocode?lat=${p.lat}&lon=${p.lon}&fields=items.full_name,items.address_name,items.name&key=${TWOGIS_KEY}`,{signal:ctrl.signal});
    clearTimeout(t);
    if(r.ok){
      const data=await r.json();
      const item=data?.result?.items?.[0];
      if(item?.full_name)return item.full_name;
      if(item?.address_name)return item.address_name;
      if(item?.name)return item.name;
    }
  }catch(e){
    console.warn('2GIS reverse error',e);
  }

  // 2. Yandex Reverse Geocoder
  if(mapAdapter?.provider==='yandex'&&window.ymaps?.geocode){
    try{
      const r=await ymaps.geocode([p.lat,p.lon]);
      const x=r.geoObjects.get(0);
      if(x?.getAddressLine())return x.getAddressLine();
    }catch(e){
      console.warn('Yandex reverse error',e);
    }
  }

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

function interests(){
  return Object.fromEntries($$('#interestPicker input').map(input=>[input.dataset.interest,Number(input.value)/10]));
}

function preferenceValues(){
  return Object.fromEntries($$('#interestPicker input').map(input=>[input.dataset.interest,Number(input.value)]));
}

function anonymousEmail(){
  let email=localStorage.getItem('routeGuestEmail');
  if(!email){
    const token=crypto.randomUUID?.()||`${Date.now()}-${Math.random().toString(16).slice(2)}`;
    email=`guest-${token}@go.local`;
    localStorage.setItem('routeGuestEmail',email);
  }
  return email;
}

async function ensureAnonymousUser(forceUpdate=false){
  const storedID=localStorage.getItem('routeUserId');
  if(storedID&&!forceUpdate){
    try{
      return await api(`/users/${storedID}`);
    }catch(error){
      localStorage.removeItem('routeUserId');
    }
  }
  const profile=await api('/users/',{
    method:'POST',
    body:JSON.stringify({name:'Путешественник',email:anonymousEmail(),interests:interests()})
  });
  localStorage.setItem('routeUserId',profile.id);
  return profile;
}

function loadPreferences(){
  let saved;
  try{
    saved=JSON.parse(localStorage.getItem('routeInterests')||'null');
  }catch(error){
    saved=null;
  }
  $$('#interestPicker input').forEach(input=>{
    if(Array.isArray(saved)){
      input.value=saved.includes(input.dataset.interest)?10:1;
    }else if(saved&&Number.isFinite(Number(saved[input.dataset.interest]))){
      input.value=Math.max(0,Math.min(10,Number(saved[input.dataset.interest])));
    }
    input.closest('label').querySelector('output').value=input.value;
  });
}

async function ensurePoints(){
  const startVal=$('#startInput').value.trim();
  let finishVal=$('#finishInput').value.trim();
  if(!startVal)throw new Error('Пожалуйста, укажите точку старта');
  if(!finishVal)finishVal=startVal;

  const [startPt,finishPt]=await Promise.all([
    geocode(startVal),
    geocode(finishVal)
  ]);

  setMarker('start',startPt,'Старт');
  state.startAddress=startVal;
  setMarker('finish',finishPt,'Финиш');
  state.finishAddress=finishVal;

  mapAdapter.fit(startPt,finishPt);
}

function esc(v){
  const e=document.createElement('div');
  e.textContent=v??'';
  return e.innerHTML;
}

function render(route){
  mapAdapter.draw(route.geojson,route.waypoints);
  $('#summaryTime').textContent=`${route.total_duration_min} мин`;
  $('#summaryDistance').textContent=`${(route.total_distance_meters/1000).toFixed(1)} км`;
  $('#routeSummary').classList.remove('hidden');
  $('#resultTitle').textContent=`Маршрут на ${route.total_duration_min} минут`;
  $('#metrics').innerHTML=`<div><strong>${route.travel_duration_min??'—'} мин</strong><small>дорога</small></div><div><strong>${route.visit_duration_min??'—'} мин</strong><small>места</small></div><div><strong>${route.arrival_buffer_min??0} мин</strong><small>запас</small></div><div><strong>≈ ${route.estimated_cost_rub??0} ₽</strong><small>стоимость</small></div>`;
  const reasons=Array.isArray(route.match_reasons)?route.match_reasons:[route.match_reasons].filter(Boolean);
  $('#reasons').innerHTML=reasons.map(x=>`<span>${esc(x)}</span>`).join('');
  $('#waypoints').innerHTML=route.waypoints.map((p,i)=>{
    const rawName=String(p.place_name||''),safeName=/^[\d\s/\\.,№#-]+$/.test(rawName)?(p.category||'Интересное место'):rawName,n=p.type==='start'?'Начало маршрута':p.type==='finish'?'Финиш':safeName,d=p.type==='place'?`${p.category} · ${p.duration_min} мин · ${(p.distance_from_prev_meters||0).toFixed(0)} м`:`${p.location.lat.toFixed(4)}, ${p.location.lon.toFixed(4)}`;
    return`<li data-order="${i+1}"><strong>${esc(n)}</strong><span>${esc(d)}</span></li>`;
  }).join('');
  form.classList.add('hidden');
  $('#results').classList.remove('hidden');
}

form.addEventListener('submit',async e=>{
  e.preventDefault();
  const b=$('#goButton');
  b.disabled=true;
  message.textContent='Ищем места и проверяем дедлайн…';
  try{
    await ensurePoints();
    const profile=await ensureAnonymousUser();
    const route=await api('/routes/build',{
      method:'POST',
      body:JSON.stringify({
        user_ids:[profile.id],
        budget_minutes:minutesBetween($('#timeFrom').value,$('#timeTo').value),
        budget_rub:Number($('#budgetInput').value)||0,
        arrival_buffer_min:Number($('#bufferInput').value)||0,
        transport_mode:state.mode,
        start:state.start,
        finish:state.finish
      })
    });
    render(route);
  }catch(err){
    console.error(err);
    message.textContent=err.message.includes('no places')||err.message.includes('no candidate')?'Места не найдены. Попробуйте выбрать другие точки или расширить время.':err.message;
  }finally{
    b.disabled=false;
  }
});

$('#editButton').addEventListener('click',()=>{
  $('#results').classList.add('hidden');
  form.classList.remove('hidden');
  mapAdapter.clearRoute?.();
});

$('#togglePlanner').addEventListener('click',()=>$('#planner').classList.toggle('collapsed'));

$$('#interestPicker input').forEach(input=>input.addEventListener('input',()=>{input.closest('label').querySelector('output').value=input.value}));
$$('#pacePicker button').forEach(b=>b.addEventListener('click',()=>{$$('#pacePicker button').forEach(x=>x.classList.remove('active'));b.classList.add('active')}));
$$('#transportPicker button').forEach(b=>b.addEventListener('click',()=>{$$('#transportPicker button').forEach(x=>x.classList.remove('active'));b.classList.add('active');state.mode=b.dataset.mode}));

$('#savePreferences').addEventListener('click',async()=>{
  const status=$('#preferencesStatus');
  localStorage.setItem('routeInterests',JSON.stringify(preferenceValues()));
  status.textContent='Сохраняем…';
  try{
    await ensureAnonymousUser(true);
    status.textContent='Готово — предпочтения сохранены';
  }catch(error){
    console.error(error);
    status.textContent='Сохранено на устройстве. Сервер сейчас недоступен.';
  }
});

$('#locateButton').addEventListener('click',()=>navigator.geolocation?.getCurrentPosition(p=>{
  const x={lat:p.coords.latitude,lon:p.coords.longitude};
  setMarker('start',x,'Вы здесь');
  reverseGeocode(x).then(a=>{$('#startInput').value=a;state.startAddress=a}).catch(()=>{});
  mapAdapter.setView(x);
},()=>$('#mapHint').textContent='Не удалось получить геопозицию'));

loadPreferences();
initMap();
