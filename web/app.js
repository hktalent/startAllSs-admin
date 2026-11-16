/* startAllSs-admin 前端逻辑 —— 纯原生，无构建 */
const TOKEN = new URLSearchParams(location.search).get('token') || '';

async function api(path, opts = {}) {
  const h = { 'Content-Type': 'application/json', ...(opts.headers || {}) };
  if (TOKEN) h['X-Admin-Token'] = TOKEN;
  const r = await fetch(path, { ...opts, headers: h });
  const t = await r.text();
  let j;
  try { j = t ? JSON.parse(t) : {}; } catch (e) { j = { raw: t }; }
  if (!r.ok) throw new Error(j.error || t || ('HTTP ' + r.status));
  return j;
}

let cfg = null;
let selIdx = 0;
let autoTimer = null;

function toast(msg, kind = '') {
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.className = 'toast ' + kind;
  el.style.display = 'block';
  clearTimeout(el._t);
  el._t = setTimeout(() => (el.style.display = 'none'), 2600);
}

/* ---------- Tabs ---------- */
document.querySelectorAll('.tab').forEach(t => {
  t.onclick = () => {
    document.querySelectorAll('.tab').forEach(x => x.classList.remove('active'));
    t.classList.add('active');
    ['svc', 'global', 'run', 'logs', 'ld'].forEach(k => {
      document.getElementById('tab-' + k).style.display = (k === t.dataset.tab) ? '' : 'none';
    });
    if (t.dataset.tab === 'logs') refreshLogs();
    if (t.dataset.tab === 'ld') refreshLaunchd();
  };
});

/* ---------- Config load ---------- */
async function loadCfg() {
  cfg = await api('/api/config');
  renderSvcList();
  if (cfg.services[selIdx]) renderEditor();
  renderGlobal();
  refreshLaunchd();
}

/* ---------- Service list ---------- */
function renderSvcList() {
  const box = document.getElementById('svc-list');
  box.innerHTML = '';
  cfg.services.forEach((s, i) => {
    const card = document.createElement('div');
    card.className = 'svc-card' + (i === selIdx ? ' active' : '');
    const pub = (s.publics || []).map(p => p.subdomain).filter(Boolean).join(',');
    card.innerHTML = `
      <div class="k">${s.key} <span class="tag">${s.type}</span>${s.builtin ? '' : ' <span class="tag">本地</span>'}</div>
      <div class="n">${s.name || '(未命名)'}</div>
      <div class="meta">📁 ${s.dir || '-'}</div>
      <div class="meta">🐳 ${(s.containers || []).join(', ') || '-'}　→　${pub || '仅本地（无域名映射）'}</div>
      <div class="meta">🔌 ${s.localScheme}://${s.localHost}:${s.localPort}${s.localHealthPath || ''}</div>`;
    card.onclick = () => { selIdx = i; renderSvcList(); renderEditor(); };
    box.appendChild(card);
  });
}

/* ---------- Service editor ---------- */
function renderEditor() {
  const s = cfg.services[selIdx];
  if (!s) return;
  document.getElementById('svc-editor').style.display = '';
  document.getElementById('ed-key').textContent = s.key;
  document.getElementById('ed-name').value = s.name || '';
  document.getElementById('ed-type').value = s.type || 'docker-single';
  document.getElementById('ed-dir').value = s.dir || '';
  document.getElementById('ed-containers').value = (s.containers || []).join(', ');
  document.getElementById('ed-command').value = s.command || '';
  document.getElementById('ed-tmux').value = s.tmuxSession || '';
  document.getElementById('ed-scheme').value = s.localScheme || 'http';
  document.getElementById('ed-host').value = s.localHost || '127.0.0.1';
  document.getElementById('ed-port').value = s.localPort || '';
  document.getElementById('ed-lpath').value = s.localHealthPath || '/';
  if (!s.publics) s.publics = [];   // 空 = 无域名映射，仅本地托管（不自动补子域名）
  renderPublics();
  renderExtras();
  document.getElementById('ed-preview').style.display = 'none';
}

function renderPublics() {
  const s = cfg.services[selIdx];
  const box = document.getElementById('ed-publics');
  box.innerHTML = '';
  s.publics.forEach((p, i) => {
    const row = document.createElement('div');
    row.className = 'subrow';
    row.innerHTML = `
      <input placeholder="子域名，如 chat" value="${p.subdomain || ''}" data-i="${i}" data-f="subdomain" style="flex:1">
      <input placeholder="探活路径，如 /" value="${p.healthPath || '/'}" data-i="${i}" data-f="healthPath" style="flex:1">
      <button class="icon-btn" data-del="${i}" title="删除">🗑</button>`;
    box.appendChild(row);
  });
  box.querySelectorAll('input').forEach(inp => {
    inp.oninput = () => {
      const i = +inp.dataset.i, f = inp.dataset.f;
      s.publics[i][f] = inp.value;
    };
  });
  box.querySelectorAll('[data-del]').forEach(b => {
    b.onclick = () => { s.publics.splice(+b.dataset.del, 1); renderPublics(); };
  });
  if (s.publics.length === 0) {
    const hint = document.createElement('div');
    hint.className = 'meta';
    hint.style.color = 'var(--muted)';
    hint.textContent = '无公网入口：纯本地服务，脚本只做拉起 / 修复 / 本地探活（不校验域名）';
    box.appendChild(hint);
  }
}

document.getElementById('ed-add-pub').onclick = () => {
  cfg.services[selIdx].publics.push({ subdomain: '', healthPath: '/' });
  renderPublics();
};

/* 新增一个无域名映射的本地服务（非内置 key，走脚本的 LOCAL_SERVICE_* 通用托管） */
document.getElementById('btn-add-svc').onclick = () => {
  const key = (prompt('服务 key（英文/数字/-，如 rcm2、nas-web）：') || '').trim();
  if (!key) return;
  if (!/^[a-zA-Z][a-zA-Z0-9_-]*$/.test(key)) return toast('key 只能以字母开头，可含数字、_ -', 'err');
  if (cfg.services.some(s => s.key === key)) return toast('key 已存在', 'err');
  cfg.services.push({
    key, name: key, type: 'docker-compose', enabled: true,
    dir: '', containers: [], command: '',
    localScheme: 'http', localHost: '127.0.0.1', localPort: 0, localHealthPath: '/',
    tmuxSession: '', publics: [],
  });
  selIdx = cfg.services.length - 1;
  renderSvcList(); renderEditor();
  toast('已新增，填好后点「保存」或「应用配置」');
};

function renderExtras() {
  const s = cfg.services[selIdx];
  const box = document.getElementById('ed-extra');
  box.innerHTML = '';
  if (s.key === 'vi') {
    box.innerHTML = `
      <div class="row">
        <div class="field" style="min-width:90px;flex:0 0 90px"><label>Qdrant 端口</label><input id="ex-qport" type="number" value="${s.qdrantPort || 6333}"></div>
        <div class="field"><label>VI_BIN_PATH</label><input id="ex-bin" value="${s.binPath || ''}"></div>
        <div class="field"><label>VI_LOG_DIR</label><input id="ex-logdir" value="${s.logDir || ''}"></div>
      </div>
      <div class="row" style="margin-top:6px">
        <div class="field"><label>VI_STOP_SCRIPT</label><input id="ex-stop" value="${s.stopScript || ''}"></div>
        <div class="field"><label>VI_START_SCRIPT</label><input id="ex-start" value="${s.startScript || ''}"></div>
      </div>`;
  } else if (s.key === 'gp') {
    box.innerHTML = `<div class="row"><div class="field" style="min-width:140px;flex:0 0 140px"><label>GP_PYTHON</label><input id="ex-py" value="${s.python || 'python3'}"></div></div>`;
  }
}

function collectEditor() {
  const s = cfg.services[selIdx];
  s.name = document.getElementById('ed-name').value;
  s.type = document.getElementById('ed-type').value;
  s.dir = document.getElementById('ed-dir').value;
  s.containers = document.getElementById('ed-containers').value.split(',').map(x => x.trim()).filter(Boolean);
  s.command = document.getElementById('ed-command').value;
  s.tmuxSession = document.getElementById('ed-tmux').value;
  s.localScheme = document.getElementById('ed-scheme').value;
  s.localHost = document.getElementById('ed-host').value;
  s.localPort = +document.getElementById('ed-port').value || 0;
  s.localHealthPath = document.getElementById('ed-lpath').value;
  if (s.key === 'vi') {
    s.qdrantPort = +document.getElementById('ex-qport')?.value || 6333;
    s.binPath = document.getElementById('ex-bin')?.value || '';
    s.logDir = document.getElementById('ex-logdir')?.value || '';
    s.stopScript = document.getElementById('ex-stop')?.value || '';
    s.startScript = document.getElementById('ex-start')?.value || '';
  }
  if (s.key === 'gp') s.python = document.getElementById('ex-py')?.value || 'python3';
}

async function saveCfg(apply = false) {
  collectEditor();
  collectGlobal();
  try {
    await api('/api/config', { method: 'PUT', body: JSON.stringify(cfg) });
    toast('已保存到 config.json', 'ok');
    if (apply) {
      const r = await api('/api/apply', { method: 'POST' });
      toast(r.ok ? '配置已应用，launchd 已重载' : '应用完成（有告警）', r.ok ? 'ok' : 'warn');
      refreshLaunchd();
    }
  } catch (e) { toast(e.message, 'err'); }
}

document.getElementById('ed-save').onclick = () => saveCfg(false);
document.getElementById('ed-apply').onclick = () => saveCfg(true);
document.getElementById('ed-render').onclick = async () => {
  collectEditor(); collectGlobal();
  try {
    await api('/api/config', { method: 'PUT', body: JSON.stringify(cfg) });
    const r = await api('/api/render');
    const pre = document.getElementById('ed-preview');
    pre.style.display = '';
    pre.textContent = r.preview;
  } catch (e) { toast(e.message, 'err'); }
};

/* ---------- Global ---------- */
const SKIP_DEFS = [
  ['stackOnly', 'STACK_ONLY 跳过 E2E_chat'],
  ['skipBuild', 'SKIP_BUILD 不自动 --build'],
  ['scrcpyOnly', 'SCRCPY_ONLY 跳过 scrcpy'],
  ['strategyOnly', 'STRATEGY_ONLY 跳过 strategy-web'],
  ['qsOnly', 'QS_ONLY 跳过 QuietShield'],
  ['viOnly', 'VI_ONLY 跳过 vi'],
  ['gpOnly', 'GP_ONLY 跳过 gp'],
  ['skipAdb', 'SKIP_ADB 跳过宿主 ADB'],
];

function renderGlobal() {
  if (!cfg) return;
  const g = cfg.global;
  document.getElementById('g-base').value = g.baseDomain || '';
  document.getElementById('g-cfc').value = g.cfContainer || '';
  document.getElementById('g-tid').value = g.tunnelId || '';
  document.getElementById('g-mode').value = g.cfMode || 'container';
  document.getElementById('g-cfg').value = g.cfConfig || '';
  document.getElementById('g-clog').value = g.cfLog || '';
  document.getElementById('g-pid').value = g.cfPidFile || '';
  document.getElementById('g-rlog').value = g.runLog || '';
  document.getElementById('g-interval').value = g.autoFixIntervalSec || 300;

  const box = document.getElementById('g-skips');
  box.innerHTML = '';
  SKIP_DEFS.forEach(([k, label]) => {
    const id = 'sk-' + k;
    const wrap = document.createElement('label');
    wrap.innerHTML = `<input type="checkbox" id="${id}"> ${label}`;
    box.appendChild(wrap);
    document.getElementById(id).checked = !!cfg.skipFlags[k];
  });
}

function collectGlobal() {
  const g = cfg.global;
  g.baseDomain = document.getElementById('g-base').value;
  g.cfContainer = document.getElementById('g-cfc').value;
  g.tunnelId = document.getElementById('g-tid').value;
  g.cfMode = document.getElementById('g-mode').value;
  g.cfConfig = document.getElementById('g-cfg').value;
  g.cfLog = document.getElementById('g-clog').value;
  g.cfPidFile = document.getElementById('g-pid').value;
  g.runLog = document.getElementById('g-rlog').value;
  g.autoFixIntervalSec = +document.getElementById('g-interval').value || 300;
  SKIP_DEFS.forEach(([k]) => {
    cfg.skipFlags[k] = document.getElementById('sk-' + k)?.checked || false;
  });
}

document.getElementById('g-save').onclick = () => saveCfg(false);
document.getElementById('g-apply').onclick = () => saveCfg(true);

/* ---------- Run ---------- */
document.querySelectorAll('[data-act]').forEach(b => {
  b.onclick = async () => {
    const act = b.dataset.act;
    const out = document.getElementById('run-out');
    out.textContent = `⏳ 执行 sh startAllSs.sh ${act} …`;
    b.disabled = true;
    try {
      const r = await api('/api/run', { method: 'POST', body: JSON.stringify({ action: act }) });
      out.textContent = r.output + (r.error ? `\n[exit] ${r.error}` : '');
      toast(r.ok ? `${act} 完成` : `${act} 退出非零`, r.ok ? 'ok' : 'warn');
    } catch (e) { out.textContent = '错误: ' + e.message; toast(e.message, 'err'); }
    finally { b.disabled = false; }
  };
});

let statusBusy = false;
document.getElementById('run-refresh').onclick = function () {
  if (autoTimer) {
    clearInterval(autoTimer); autoTimer = null;
    this.textContent = '⟳ 自动刷新'; toast('已停止自动刷新');
    return;
  }
  const tick = async () => {
    if (statusBusy) return;
    statusBusy = true;
    try {
      const r = await api('/api/status');
      document.getElementById('run-out').textContent = r.output;
    } catch (e) { /* ignore */ }
    finally { statusBusy = false; }
  };
  tick();
  autoTimer = setInterval(tick, 30000);
  this.textContent = '⏹ 停止刷新';
  toast('每 30s 刷新状态');
};

/* ---------- Logs ---------- */
async function refreshLogs() {
  const f = document.getElementById('log-file').value;
  const n = +document.getElementById('log-lines').value || 200;
  try {
    const r = await api(`/api/logs?file=${f}&lines=${n}`);
    document.getElementById('log-out').textContent = r.output || (r.error ? '(空或读取失败: ' + r.error + ')' : '(空)');
  } catch (e) { document.getElementById('log-out').textContent = '错误: ' + e.message; }
}
document.getElementById('log-refresh').onclick = refreshLogs;
document.getElementById('log-file').onchange = refreshLogs;
document.getElementById('log-lines').onchange = refreshLogs;

/* ---------- Launchd ---------- */
async function refreshLaunchd() {
  try {
    const r = await api('/api/launchd');
    const info = r.info || {};
    const state = info.state || 'unknown';
    const dot = document.getElementById('ld-dot');
    dot.className = 'dot ' + (state === 'running' ? 'ok' : (state === 'not running' ? '' : 'warn'));
    document.getElementById('ld-state').textContent = state;
    document.getElementById('ld-interval').textContent = info.startInterval || '?';
    document.getElementById('ld-runs').textContent = info.runs || '?';

    const kv = document.getElementById('ld-kv');
    kv.innerHTML = '';
    const rows = [
      ['Label', info.label || 'com.51pwn.startallss'],
      ['State', info.state],
      ['Runs', info.runs],
      ['Last exit code', info.lastExitCode],
      ['Active count', info.activeCount],
      ['StartInterval', info.startInterval + 's'],
      ['target', r.info.target || ''],
    ];
    rows.forEach(([k, v]) => {
      kv.insertAdjacentHTML('beforeend', `<div class="k">${k}</div><div>${v ?? ''}</div>`);
    });
    if (info.raw) {
      kv.insertAdjacentHTML('beforeend', `<div class="k" style="grid-column:1/-1;margin-top:6px">launchctl print（原始）</div>`);
      kv.insertAdjacentHTML('beforeend', `<pre class="out" style="grid-column:1/-1">${info.raw}</pre>`);
    }
  } catch (e) { /* header may be offline */ }
}

document.getElementById('btn-kick').onclick = async () => {
  try { const r = await api('/api/launchd?action=kickstart', { method: 'POST' }); toast('已触发: ' + (r.log || ''), r.ok ? 'ok' : 'err'); }
  catch (e) { toast(e.message, 'err'); }
};
document.getElementById('btn-reload').onclick = async () => {
  try { const r = await api('/api/launchd?action=reload', { method: 'POST' }); toast('重载: ' + (r.log || ''), r.ok ? 'ok' : 'err'); refreshLaunchd(); }
  catch (e) { toast(e.message, 'err'); }
};
document.getElementById('ld-kick2').onclick = document.getElementById('btn-kick').onclick;
document.getElementById('ld-reload2').onclick = document.getElementById('btn-reload').onclick;

/* ---------- 启动 ---------- */
loadCfg().catch(e => toast('加载配置失败: ' + e.message, 'err'));
setInterval(refreshLaunchd, 10000);
