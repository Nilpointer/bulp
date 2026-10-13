'use strict';

const $ = (sel, root = document) => root.querySelector(sel);
const list = $('#list');
const rows = new Map();     // id -> { el, light }
const pending = new Map();  // id -> patch waiting to be sent
const inflight = new Set(); // ids with a PUT in progress
let dragging = null;        // slider currently held by the user
let pollTimer = null;

async function api(path, opts = {}) {
  const r = await fetch(path, { headers: { 'Content-Type': 'application/json' }, ...opts });
  if (!r.ok) throw new Error((await r.text()).trim() || r.statusText);
  return r.status === 204 ? null : r.json();
}

/* ---------- rendering ---------- */

const kelvin = mired => Math.round(1e6 / mired / 50) * 50;
const isOn = l => l.state && l.state.on === 1;

function wifiLevel(rssi) {
  if (rssi == null) return 0;
  return rssi >= -55 ? 4 : rssi >= -67 ? 3 : rssi >= -78 ? 2 : 1;
}

function render(lights) {
  const seen = new Set();
  for (const l of lights) {
    seen.add(l.id);
    let row = rows.get(l.id);
    if (!row) {
      row = { el: createRow(l.id) };
      rows.set(l.id, row);
    }
    row.light = l;
    list.appendChild(row.el); // keeps server order (sorted by name)
    updateRow(row);
  }
  for (const [id, row] of rows) {
    if (!seen.has(id)) { row.el.remove(); rows.delete(id); }
  }
  $('#empty').hidden = lights.length > 0;
  if (!$('#rescan').classList.contains('spinning')) $('#empty-title').textContent = 'No Accessories Found';
  $('#all-power').classList.toggle('on', lights.some(isOn));
  resize();
}

function createRow(id) {
  const el = $('#light-tpl').content.firstElementChild.cloneNode(true);

  $('.power', el).onclick = () => {
    const l = rows.get(id).light;
    send(id, { on: isOn(l) ? 0 : 1 });
  };
  $('.settings', el).onclick = () => { $('.details', el).hidden ^= 1; resize(); };
  $('.identify', el).onclick = () => api(`api/lights/${encodeURIComponent(id)}/identify`, { method: 'POST' }).catch(showError);

  const bind = (cls, key, label) => {
    const s = $(cls, el);
    s.addEventListener('pointerdown', () => { dragging = s; showBubble(s, label(+s.value)); });
    s.addEventListener('input', () => {
      paintFill(s);
      showBubble(s, label(+s.value));
      if (key === 'hue') el.style.setProperty('--hue-color', `hsl(${s.value} 100% 50%)`);
      send(id, { [key]: +s.value });
    });
    const release = () => { if (dragging === s) dragging = null; hideBubble(); };
    s.addEventListener('pointerup', release);
    s.addEventListener('pointercancel', release);
    s.addEventListener('change', release);
    return s;
  };
  bind('.bright-slider', 'brightness', v => `${v}%`);
  bind('.temp-slider', 'temperature', v => `${kelvin(v)} K`);
  bind('.hue-slider', 'hue', v => `${v}°`);
  bind('.sat-slider', 'saturation', v => `${v}%`);

  // −/+ buttons next to each slider, like the original's icon buttons
  for (const b of el.querySelectorAll('.step')) {
    b.onclick = () => {
      const s = $('.slider', b.parentElement);
      s.value = Math.min(+s.max, Math.max(+s.min, +s.value + +b.dataset.step));
      s.dispatchEvent(new Event('input'));
      hideBubble();
    };
  }
  return el;
}

function updateRow({ el, light: l }) {
  const st = l.state || {};
  const busy = pending.has(l.id) || inflight.has(l.id);

  $('.name', el).textContent = l.name;
  el.classList.toggle('off', !isOn(l));
  $('.power', el).classList.toggle('on', isOn(l));
  $('.power', el).disabled = !l.state;

  const lvl = wifiLevel(l.rssi);
  $('.wifi', el).toggleAttribute('hidden', !lvl); // SVG elements have no .hidden property
  if (lvl) {
    $('.wifi use', el).setAttribute('href', `#i-wifi-${lvl}`);
    $('.wifi', el).setAttribute('aria-label', `Wi-Fi ${l.rssi} dBm`);
  }

  const isColor = st.hue != null;
  $('.temp', el).hidden = isColor || st.temperature == null;
  $('.hue', el).hidden = !isColor;
  $('.sat', el).hidden = !isColor;
  $('.bright', el).hidden = st.brightness == null;

  // Don't fight the user: skip slider updates while dragging or while our own change is in flight.
  if (!busy) {
    setSlider($('.bright-slider', el), st.brightness);
    setSlider($('.temp-slider', el), st.temperature);
    setSlider($('.hue-slider', el), st.hue);
    setSlider($('.sat-slider', el), st.saturation);
    if (isColor) el.style.setProperty('--hue-color', `hsl(${st.hue} 100% 50%)`);
  }

  $('.d-product', el).textContent = l.product || '—';
  $('.d-firmware', el).textContent = l.firmware || '—';
  $('.d-serial', el).textContent = l.serial || '—';
  $('.d-host', el).textContent = l.host;
  $('.d-wifi', el).textContent = l.rssi != null ? `${l.rssi} dBm` : '—';

  $('.error', el).hidden = !l.error;
  $('.error', el).textContent = l.error ? `Not reachable: ${l.error}` : '';
}

function setSlider(s, v) {
  if (v == null || s === dragging) return;
  s.value = v;
  paintFill(s);
}

function paintFill(s) {
  const p = (s.value - s.min) / (s.max - s.min) * 100;
  s.style.setProperty('--fill', `${p}%`);
}

/* ---------- value bubble (SliderTooltipManager in the original) ---------- */

const bubble = $('#bubble');
function showBubble(s, text) {
  const r = s.getBoundingClientRect();
  const p = (s.value - s.min) / (s.max - s.min);
  bubble.textContent = text;
  bubble.style.left = `${r.left + 8 + p * (r.width - 16)}px`;
  bubble.style.top = `${r.top - 2}px`;
  bubble.hidden = false;
}
function hideBubble() { bubble.hidden = true; }

/* ---------- sending changes: one request in flight per light, latest value wins ---------- */

function send(id, patch) {
  pending.set(id, { ...(pending.get(id) || {}), ...patch });
  pump(id);
}

async function pump(id) {
  if (inflight.has(id) || !pending.has(id)) return;
  const body = pending.get(id);
  pending.delete(id);
  inflight.add(id);
  try {
    const st = await api(`api/lights/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(body) });
    const row = rows.get(id);
    if (row) {
      row.light = { ...row.light, state: st, error: '' };
      inflight.delete(id);
      updateRow(row);
      $('#all-power').classList.toggle('on', [...rows.values()].some(r => isOn(r.light)));
    }
  } catch (e) {
    showError(e);
  } finally {
    inflight.delete(id);
    pump(id);
  }
}

/* ---------- top bar ---------- */

$('#all-power').onclick = () => {
  const target = [...rows.values()].some(r => isOn(r.light)) ? 0 : 1;
  for (const id of rows.keys()) send(id, { on: target });
};

async function rescan() {
  $('#rescan').classList.add('spinning');
  $('#empty-title').textContent = 'Searching…';
  try { render(await api('api/rescan', { method: 'POST' })); }
  catch (e) { showError(e); }
  finally {
    $('#rescan').classList.remove('spinning');
    $('#empty-title').textContent = 'No Accessories Found';
  }
}
$('#rescan').onclick = rescan;
$('#empty-rescan').onclick = rescan;

const menu = $('#menu');
$('#menu-btn').onclick = e => { e.stopPropagation(); menu.hidden = !menu.hidden; };
document.addEventListener('click', () => { menu.hidden = true; });
menu.onclick = e => {
  const a = e.target.dataset.action;
  if (a === 'rescan') rescan();
  if (a === 'quit') api('api/quit', { method: 'POST' }).catch(() => {}).finally(() => window.close());
};

/* ---------- refresh, errors, sizing ---------- */

async function refresh() {
  try { render(await api('api/lights')); }
  catch (e) { showError(e); }
}

let toastTimer;
function showError(e) {
  const t = $('#toast');
  t.textContent = e.message || String(e);
  t.hidden = false;
  resize();
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { t.hidden = true; resize(); }, 4000);
}

function resize() {
  requestAnimationFrame(() => {
    const h = document.body.scrollHeight;
    window.webkit?.messageHandlers?.resize?.postMessage(h);
  });
}

// Called by the Cocoa side when the popover opens/closes.
window.onPopoverShow = () => {
  refresh();
  clearInterval(pollTimer);
  pollTimer = setInterval(refresh, 4000);
};
window.onPopoverHide = () => {
  clearInterval(pollTimer);
  menu.hidden = true;
  for (const row of rows.values()) $('.details', row.el).hidden = true;
};

// Outside the macOS popover (Linux app window, plain browser) there are no
// show/hide callbacks: poll while the page is visible instead.
const inPopover = !!window.webkit?.messageHandlers?.resize;
if (inPopover) {
  refresh();
} else {
  document.documentElement.classList.add('standalone');
  const sync = () => (document.hidden ? window.onPopoverHide() : window.onPopoverShow());
  document.addEventListener('visibilitychange', sync);
  sync();
  // Lets the app exit with its window when there's no tray icon to keep it around.
  window.addEventListener('pagehide', () => navigator.sendBeacon('api/closed'));
}
