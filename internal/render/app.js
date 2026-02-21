// Build reverse index: import name → [files]
const reverseIndex = {};
data.forEach(f => f.imports.forEach(i => {
  (reverseIndex[i.name] ??= []).push(f.file);
}));

// Build depended-on count per file (how many other files import it)
const dependedOn = {};
const fileSet = new Set(data.map(f => f.file));
data.forEach(f => f.imports.forEach(i => {
  if (i.category !== 'internal') return;
  const norm = i.name.replace(/^\.\.?\//, '');
  for (const file of fileSet) {
    if (file.includes(norm)) { dependedOn[file] = (dependedOn[file] || 0) + 1; break; }
  }
}));

// Category counts
const catCounts = { stdlib: 0, internal: 0, private: 0, external: 0 };
data.forEach(f => f.imports.forEach(i => catCounts[i.category]++));
Object.keys(catCounts).forEach(c => {
  const el = document.getElementById('count-' + c);
  if (el) el.textContent = catCounts[c];
});

// Stats
const allNames = data.flatMap(f => f.imports.map(i => i.name));
const totalExports = data.reduce((n, f) => n + (f.exports || []).length, 0);
const avg = data.length ? (allNames.length / data.length).toFixed(1) : 0;
document.getElementById('stat-files').textContent = data.length + ' files';
document.getElementById('stat-imports').textContent = new Set(allNames).size + ' unique imports';
document.getElementById('stat-exports').textContent = totalExports + ' exports';
document.getElementById('stat-avg').textContent = avg + ' avg imports/file';
// Line count & language breakdown
const totalLines = data.reduce((n, f) => n + (f.lines || 0), 0);
document.getElementById('stat-lines').textContent = totalLines.toLocaleString() + ' total lines';
const langMap = {};
data.forEach(f => {
  const ext = f.file.substring(f.file.lastIndexOf('.'));
  langMap[ext] = (langMap[ext] || 0) + (f.lines || 0);
});
const langs = Object.entries(langMap).sort((a, b) => b[1] - a[1]);
const langColors = { '.ts': '#3178c6', '.tsx': '#61dafb', '.js': '#f7df1e', '.jsx': '#61dafb', '.mjs': '#f7df1e', '.go': '#00add8', '.css': '#563d7c', '.scss': '#c6538c', '.html': '#e34c26', '.json': '#a8a8a8', '.md': '#555', '.yml': '#cb171e', '.yaml': '#cb171e' };
const langNames = { '.ts': 'TypeScript', '.tsx': 'TSX', '.js': 'JavaScript', '.jsx': 'JSX', '.mjs': 'JavaScript', '.go': 'Go', '.css': 'CSS', '.scss': 'SCSS', '.html': 'HTML', '.json': 'JSON', '.md': 'Markdown', '.yml': 'YAML', '.yaml': 'YAML' };
document.getElementById('lang-bar').innerHTML = langs.map(([ext, lines]) => {
  const pct = (lines / totalLines * 100).toFixed(1);
  const col = langColors[ext] || '#8b949e';
  return '<span style="width:' + pct + '%;background:' + col + '" title="' + (langNames[ext] || ext) + ' ' + pct + '%"></span>';
}).join('');
document.getElementById('lang-legend').innerHTML = langs.map(([ext, lines]) => {
  const pct = (lines / totalLines * 100).toFixed(0);
  const col = langColors[ext] || '#8b949e';
  return '<span style="color:' + col + '">' + (langNames[ext] || ext) + ' ' + pct + '%</span>';
}).join('');
// Top 5 most imported
const freq = {};
allNames.forEach(n => freq[n] = (freq[n] || 0) + 1);
const top5 = Object.entries(freq).sort((a, b) => b[1] - a[1]).slice(0, 5);
document.getElementById('top-imports').innerHTML = top5.map(([name, count]) =>
  '<li title="' + name + '"><span>' + name + '</span><span class="ti-count">' + count + '</span></li>'
).join('');
// Category breakdown bar
const total = allNames.length || 1;
const catColors = { stdlib: 'var(--green)', internal: 'var(--purple)', private: 'var(--blue)', external: 'var(--orange)' };
document.getElementById('cat-bar').innerHTML = ['stdlib','internal','private','external'].map(c => {
  const pct = (catCounts[c] / total * 100).toFixed(1);
  return '<span style="width:' + pct + '%;background:' + catColors[c] + '"></span>';
}).join('');
document.getElementById('cat-bar-legend').innerHTML = ['stdlib','internal','private','external'].map(c => {
  const pct = (catCounts[c] / total * 100).toFixed(0);
  return '<span style="color:' + catColors[c] + '">' + c + ' ' + pct + '%</span>';
}).join('');
// God files (10+ imports)
const godFiles = data.filter(f => f.imports.length >= 10).sort((a, b) => b.imports.length - a.imports.length).slice(0, 5);
const gfEl = document.getElementById('god-files');
const gfSection = document.getElementById('god-files-section');
if (godFiles.length) {
  gfEl.innerHTML = godFiles.map(f =>
    '<li title="' + f.file + '"><span>' + f.file + '</span><span class="ti-count">' + f.imports.length + '</span></li>'
  ).join('');
} else { gfSection.style.display = 'none'; gfEl.style.display = 'none'; }
document.getElementById('root-path').textContent = root;

// State
const active = new Set(['stdlib', 'internal', 'private', 'external']);
let selectedImport = null;
const collapsedFiles = new Set();

// URL hash state
function readHash() {
  const p = new URLSearchParams(location.hash.slice(1));
  if (p.has('q')) searchInput.value = p.get('q');
  if (p.has('view')) {
    document.querySelectorAll('.view-btn').forEach(b => b.classList.toggle('active', b.dataset.view === p.get('view')));
  }
  if (p.has('sort')) document.getElementById('sort').value = p.get('sort');
  if (p.has('cats')) {
    const cats = new Set(p.get('cats').split(','));
    active.clear();
    cats.forEach(c => active.add(c));
    document.querySelectorAll('.filter-btn').forEach(b => b.classList.toggle('active', active.has(b.dataset.cat)));
  }
  if (p.has('rev')) {
    const rev = p.get('rev');
    if (reverseIndex[rev]) showReverse(rev);
  }
}
function writeHash() {
  const p = new URLSearchParams();
  const q = searchInput.value;
  if (q) p.set('q', q);
  const view = document.querySelector('.view-btn.active').dataset.view;
  if (view !== 'both') p.set('view', view);
  const sort = document.getElementById('sort').value;
  if (sort !== 'name-asc') p.set('sort', sort);
  const cats = [...active].sort().join(',');
  if (cats !== 'external,internal,private,stdlib') p.set('cats', cats);
  if (selectedImport) p.set('rev', selectedImport);
  const h = p.toString();
  history.replaceState(null, '', h ? '#' + h : location.pathname);
}

// Filter buttons
document.querySelectorAll('.filter-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    const cat = btn.dataset.cat;
    btn.classList.toggle('active');
    active.has(cat) ? active.delete(cat) : active.add(cat);
    render();
  });
});

// Sort
document.getElementById('sort').addEventListener('change', render);
document.querySelectorAll('.view-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('.view-btn').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    render();
  });
});

// Search
const searchInput = document.getElementById('search');
let debounceTimer;
searchInput.addEventListener('input', () => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => { selectedImport = null; render(); }, 150);
});

// Reverse panel close
document.getElementById('reverse-close').addEventListener('click', () => {
  selectedImport = null;
  document.getElementById('code-panel').classList.remove('visible');
  render();
});

function showReverse(importName) {
  selectedImport = importName;
  const panel = document.getElementById('reverse-panel');
  const files = reverseIndex[importName] || [];
  document.getElementById('reverse-title').textContent = importName;
  document.getElementById('reverse-count').textContent = files.length + ' file' + (files.length !== 1 ? 's' : '') + ' use this import';
  const list = document.getElementById('reverse-list');
  list.innerHTML = files.map(f =>
    '<li><a href="vscode://file/' + root + '/' + f + '">' + f + '</a></li>'
  ).join('');
  panel.classList.add('visible');
  render();
}

function escHtml(s) {
  return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}

function highlightSyntax(code) {
  const kw = /\b(import|export|from|as|const|let|var|require|type|default|await)\b/g;
  const str = /(["'`])(?:[^"'`\\]|\\.)*?\1/g;
  const punct = /[{}(),;*]/g;
  const token = new RegExp('(' + kw.source + ')|(' + str.source + ')|(' + punct.source + ')', 'g');
  return code.replace(token, (m, kwM, _, strM, punctM) => {
    if (kwM) return '<span class="kw">' + escHtml(kwM) + '</span>';
    if (strM) return '<span class="str">' + escHtml(strM) + '</span>';
    if (punctM) return '<span class="punct">' + escHtml(punctM) + '</span>';
    return escHtml(m);
  });
}

// Build snippet lookup: "file::importName" → {snippet, kind, line}
const snippetIndex = {};
data.forEach(f => f.imports.forEach(i => {
  if (i.snippet) snippetIndex[f.file + '::' + i.name] = { snippet: i.snippet, kind: i.kind, line: i.line };
}));

function showCode(file, importName, clickEvent) {
  const panel = document.getElementById('code-panel');
  const entry = snippetIndex[file + '::' + importName];
  if (!entry || !entry.snippet) { panel.classList.remove('visible'); return; }
  document.getElementById('code-title').textContent = importName;
  document.getElementById('code-kind').textContent = entry.kind || '';
  document.getElementById('code-body').innerHTML = highlightSyntax(entry.snippet);
  const link = document.getElementById('code-link');
  const lineRef = entry.line ? ':' + entry.line : '';
  link.href = 'vscode://file/' + root + '/' + file + lineRef;
  link.textContent = file + (entry.line ? ':' + entry.line : '');
  const usages = document.getElementById('code-usages');
  const files = reverseIndex[importName] || [];
  if (files.length > 0) {
    usages.textContent = files.length + ' file' + (files.length !== 1 ? 's' : '') + ' use this import';
    usages.onclick = () => showReverse(importName);
    usages.style.display = '';
  } else {
    usages.style.display = 'none';
  }
  // Position near click
  if (clickEvent) {
    const x = Math.min(clickEvent.clientX, window.innerWidth - 500);
    const y = Math.min(clickEvent.clientY + 10, window.innerHeight - 300);
    panel.style.left = Math.max(0, x) + 'px';
    panel.style.top = Math.max(0, y) + 'px';
  }
  panel.classList.add('visible');
}

document.getElementById('code-close').addEventListener('click', () => {
  document.getElementById('code-panel').classList.remove('visible');
});

// Copy snippet to clipboard
document.getElementById('code-copy').addEventListener('click', () => {
  const code = document.getElementById('code-body').textContent;
  navigator.clipboard.writeText(code).then(() => {
    const btn = document.getElementById('code-copy');
    btn.textContent = '✓';
    setTimeout(() => btn.textContent = '⎘', 1500);
  });
});

// Keyboard shortcuts
document.addEventListener('keydown', e => {
  if (e.key === 'Escape') {
    document.getElementById('code-panel').classList.remove('visible');
    document.getElementById('reverse-panel').classList.remove('visible');
    selectedImport = null;
    render();
  }
  if (e.key === '/' && document.activeElement !== searchInput) {
    e.preventDefault();
    searchInput.focus();
  }
});

// File icon map (Devicon classes)
const nameIcons = [
  [/vite\.config/, 'devicon-vitejs-plain'],
  [/tailwind\.config/, 'devicon-tailwindcss-original'],
  [/jest\.config|jest\.setup/, 'devicon-jest-plain'],
  [/webpack\.config/, 'devicon-webpack-plain'],
  [/babel\.config|\.babelrc/, 'devicon-babel-plain'],
  [/\.eslintrc|eslint\.config/, 'devicon-eslint-original'],
  [/next\.config/, 'devicon-nextjs-plain'],
  [/nuxt\.config/, 'devicon-nuxtjs-plain'],
  [/Dockerfile|\.dockerignore/, 'devicon-docker-plain'],
  [/\.github/, 'devicon-github-original'],
  [/package\.json/, 'devicon-npm-original-wordmark'],
];
const extIcons = {
  '.tsx': 'devicon-react-original', '.jsx': 'devicon-react-original',
  '.ts': 'devicon-typescript-plain', '.js': 'devicon-javascript-plain', '.mjs': 'devicon-javascript-plain',
  '.go': 'devicon-go-original-wordmark',
  '.css': 'devicon-css3-plain', '.scss': 'devicon-sass-original',
  '.json': 'devicon-json-plain', '.md': 'devicon-markdown-original',
  '.html': 'devicon-html5-plain', '.yml': 'devicon-yaml-plain', '.yaml': 'devicon-yaml-plain',
};
function fileIcon(path) {
  const name = path.substring(path.lastIndexOf('/') + 1);
  for (const [re, cls] of nameIcons) { if (re.test(name)) return '<i class="' + cls + ' colored"></i>'; }
  const ext = name.substring(name.lastIndexOf('.'));
  const cls = extIcons[ext];
  return cls ? '<i class="' + cls + ' colored"></i>' : '📄';
}

// File tree
(function() {
  const tree = {};
  data.forEach(f => {
    const parts = f.file.split('/');
    let node = tree;
    parts.forEach((p, i) => {
      if (i === parts.length - 1) { (node.__files ??= []).push(f.file); }
      else { node[p] ??= {}; node = node[p]; }
    });
  });
  const el = document.getElementById('file-tree');
  function renderDir(obj, depth) {
    let html = '';
    const dirs = Object.keys(obj).filter(k => k !== '__files').sort();
    const files = (obj.__files || []).sort();
    dirs.forEach(d => {
      const count = countFiles(obj[d]);
      html += '<div class="ft-dir" data-depth="' + depth + '" style="padding-left:' + (0.5 + depth * 0.75) + 'rem">' +
        '<span class="ft-chevron">▾</span><span class="ft-label">📁 ' + d + '</span><span class="ft-count">' + count + '</span></div>' +
        '<div class="ft-children">' + renderDir(obj[d], depth + 1) + '</div>';
    });
    files.forEach(f => {
      const name = f.substring(f.lastIndexOf('/') + 1);
      html += '<div class="ft-file" data-file="' + f + '" style="padding-left:' + (0.5 + depth * 0.75 + 0.8) + 'rem">' +
        '<span class="ft-label">' + fileIcon(f) + ' ' + name + '</span></div>';
    });
    return html;
  }
  function countFiles(obj) {
    let n = (obj.__files || []).length;
    Object.keys(obj).filter(k => k !== '__files').forEach(k => n += countFiles(obj[k]));
    return n;
  }
  el.innerHTML = renderDir(tree, 0);
  el.addEventListener('click', e => {
    const dir = e.target.closest('.ft-dir');
    if (dir) {
      const children = dir.nextElementSibling;
      if (children && children.classList.contains('ft-children')) {
        children.classList.toggle('collapsed');
        dir.querySelector('.ft-chevron').classList.toggle('collapsed');
      }
      return;
    }
    const file = e.target.closest('.ft-file');
    if (file) {
      const card = document.querySelector('.card[data-file="' + file.dataset.file + '"]');
      if (card) { card.scrollIntoView({ behavior: 'smooth', block: 'center' }); card.classList.add('highlighted'); setTimeout(() => card.classList.remove('highlighted'), 1500); }
      el.querySelectorAll('.ft-file').forEach(f => f.classList.remove('active'));
      file.classList.add('active');
    }
  });
})();

function sortData(items) {
  const mode = document.getElementById('sort').value;
  const sorted = [...items];
  switch (mode) {
    case 'name-asc': return sorted.sort((a, b) => a.file.localeCompare(b.file));
    case 'name-desc': return sorted.sort((a, b) => b.file.localeCompare(a.file));
    case 'imports-desc': return sorted.sort((a, b) => b.imports.length - a.imports.length);
    case 'imports-asc': return sorted.sort((a, b) => a.imports.length - b.imports.length);
    case 'depended-desc': return sorted.sort((a, b) => (dependedOn[b.file] || 0) - (dependedOn[a.file] || 0));
    default: return sorted;
  }
}

function render() {
  const q = searchInput.value.toLowerCase();
  const viewMode = document.querySelector('.view-btn.active').dataset.view;
  const grid = document.getElementById('grid');
  grid.innerHTML = '';
  let shown = 0;

  // If reverse lookup active and no search, filter to files using that import
  const reverseFiles = selectedImport ? new Set(reverseIndex[selectedImport] || []) : null;

  const sorted = sortData(data);

  sorted.forEach(f => {
    const visibleImports = f.imports.filter(i => active.has(i.category));
    const exports = f.exports || [];
    const showImports = viewMode !== 'exports';
    const showExports = viewMode !== 'imports';

    const hasContent = (showImports && visibleImports.length > 0) || (showExports && exports.length > 0);
    if (!hasContent) return;

    if (reverseFiles && !reverseFiles.has(f.file)) return;

    const fileMatch = f.file.toLowerCase().includes(q);
    const importMatch = visibleImports.some(i => i.name.toLowerCase().includes(q));
    const exportMatch = exports.some(e => e.name.toLowerCase().includes(q));
    if (q && !fileMatch && !importMatch && !exportMatch) return;

    const card = document.createElement('div');
    const isHighlighted = reverseFiles && reverseFiles.has(f.file);
    card.className = 'card' + (isHighlighted ? ' highlighted' : '');
    card.dataset.file = f.file;

    const tags = visibleImports.map(i => {
      const isSelected = selectedImport === i.name;
      const highlight = q && i.name.toLowerCase().includes(q);
      let cls = 'tag tag-' + i.category;
      if (isSelected) cls += ' selected';
      let style = highlight ? ' style="outline:1px solid var(--accent)"' : '';

      let detail = '';
      if (i.kind) {
        let lines = '<span class="detail-kind">' + i.kind + '</span>';
        if (i.alias) lines += '<div class="detail-names">as ' + i.alias + '</div>';
        if (i.names && i.names.length) lines += '<div class="detail-names">{ ' + i.names.join(', ') + ' }</div>';
        detail = '<span class="tag-detail">' + lines + '</span>';
      }

      return '<span class="' + cls + '"' + style + ' data-import="' + i.name + '" data-kind="' + (i.kind || '') + '">' + i.name + detail + '</span>';
    }).join('');

    const exportTags = exports.map(e => {
      const highlight = q && e.name.toLowerCase().includes(q);
      let cls = 'etag';
      if (e.private) cls += ' private';
      let style = highlight ? ' style="outline:1px solid var(--accent)"' : '';
      const lineAttr = e.line ? ' data-line="' + e.line + '"' : '';
      const tooltip = e.line ? '<span class="etag-detail">line ' + e.line + '</span>' : '';
      return '<span class="' + cls + '"' + style + ' data-file="' + f.file + '"' + lineAttr + '>' + e.name + '<span class="ekind">' + e.kind + '</span>' + tooltip + '</span>';
    }).join('');

    const count = (showImports ? visibleImports.length : 0) + (showExports ? exports.length : 0);

    var sections = '';
    if (showImports) {
      const content = tags || '<span class="section-empty">no imports</span>';
      sections += '<div class="card-section"><div class="card-section-label">Imports (' + visibleImports.length + ')</div><div class="tags">' + content + '</div></div>';
    }
    if (showExports) {
      const content = exportTags || '<span class="section-empty">no exports</span>';
      sections += '<div class="card-section"><div class="card-section-label">Exports (' + exports.length + ')</div><div class="export-row">' + content + '</div></div>';
    }

    card.innerHTML =
      '<div class="card-header">' +
        '<div class="file-info">' +
          '<span class="file-icon">' + fileIcon(f.file) + '</span>' +
          '<a href="vscode://file/' + root + '/' + f.file + '">' + f.file + '</a>' +
        '</div>' +
        '<div class="header-right">' +
          '<span class="import-count">' + count + '</span>' +
          '<button class="collapse-btn" title="Collapse">▾</button>' +
        '</div>' +
      '</div>' + sections;

    if (collapsedFiles.has(f.file)) card.classList.add('collapsed');

    grid.appendChild(card);
    shown++;
  });

  // Update reverse panel visibility
  const panel = document.getElementById('reverse-panel');
  if (!selectedImport) panel.classList.remove('visible');

  document.getElementById('result-count').textContent = shown + ' of ' + data.length + ' files';
  document.getElementById('no-results').style.display = shown === 0 ? 'block' : 'none';
  writeHash();
}

// Event delegation on grid
grid.addEventListener('click', e => {
  const tag = e.target.closest('.tag');
  if (tag) {
    const card = tag.closest('.card');
    showCode(card.dataset.file, tag.dataset.import, e);
    return;
  }
  const etag = e.target.closest('.etag');
  if (etag) {
    const lineRef = etag.dataset.line ? ':' + etag.dataset.line : '';
    window.open('vscode://file/' + root + '/' + etag.dataset.file + lineRef, '_self');
    return;
  }
  const header = e.target.closest('.card-header');
  if (header && !e.target.closest('a')) {
    const card = header.closest('.card');
    card.classList.toggle('collapsed');
    const file = card.dataset.file;
    if (card.classList.contains('collapsed')) { collapsedFiles.add(file); } else { collapsedFiles.delete(file); }
  }
});

readHash();
render();

// Mobile sidebar toggle
const menuToggle = document.getElementById('menu-toggle');
const sidebar = document.getElementById('sidebar');
const overlay = document.getElementById('sidebar-overlay');
function toggleSidebar() {
  sidebar.classList.toggle('open');
  overlay.classList.toggle('visible');
}
menuToggle.addEventListener('click', toggleSidebar);
overlay.addEventListener('click', toggleSidebar);

// Theme select
const themeSelect = document.getElementById('theme-select');
function applyTheme(theme) {
  document.documentElement.setAttribute('data-theme', theme);
  themeSelect.value = theme;
  localStorage.setItem('depviz-theme', theme);
}
themeSelect.addEventListener('change', () => applyTheme(themeSelect.value));
const saved = localStorage.getItem('depviz-theme');
if (saved) applyTheme(saved);
