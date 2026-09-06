document.addEventListener('DOMContentLoaded', () => {
  let state = {
    current: null,
    palette: null,
    wallpapers: [],
    config: null,
    activeTopicFilter: '',
    searchQuery: '',
    activeExportTab: 'hyprland'
  };

  // Elements
  const currentThumb = document.getElementById('currentThumb');
  const currentTitle = document.getElementById('currentTitle');
  const currentTopic = document.getElementById('currentTopic');
  const currentDim = document.getElementById('currentDim');
  const currentMode = document.getElementById('currentMode');
  const ambientGlow = document.getElementById('ambientGlow');
  const swatchesContainer = document.getElementById('swatchesContainer');
  const ansiContainer = document.getElementById('ansiContainer');
  const wallpaperGrid = document.getElementById('wallpaperGrid');
  const wallpaperCount = document.getElementById('wallpaperCount');
  const topicFilters = document.getElementById('topicFilters');
  const searchInput = document.getElementById('searchInput');
  const colorCardsContainer = document.getElementById('colorCardsContainer');
  const codeBlock = document.getElementById('codeBlock');
  const toast = document.getElementById('toast');

  const btnRotate = document.getElementById('btnRotate');
  const btnSync = document.getElementById('btnSync');
  const btnSaveConfig = document.getElementById('btnSaveConfig');
  const saveStatus = document.getElementById('saveStatus');

  // Tab switching
  document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
      document.querySelectorAll('.tab-pane').forEach(p => p.classList.remove('active'));
      btn.classList.add('active');
      const tabId = `tab-${btn.dataset.tab}`;
      document.getElementById(tabId)?.classList.add('active');
    });
  });

  // Export file preview tabs
  document.querySelectorAll('.exp-tab').forEach(btn => {
    btn.addEventListener('click', () => {
      document.querySelectorAll('.exp-tab').forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      state.activeExportTab = btn.dataset.file;
      renderCodePreview();
    });
  });

  // Search input
  searchInput.addEventListener('input', (e) => {
    state.searchQuery = e.target.value.toLowerCase();
    renderWallpaperGrid();
  });

  // Toast notification
  function showToast(msg) {
    toast.textContent = msg;
    toast.classList.add('show');
    setTimeout(() => toast.classList.remove('show'), 2600);
  }

  // Copy to clipboard
  function copyToClipboard(text, label) {
    navigator.clipboard.writeText(text).then(() => {
      showToast(`Copied ${label || text} to clipboard!`);
    }).catch(() => {
      showToast(`Value: ${text}`);
    });
  }

  // Fetch API helpers
  async function apiGet(endpoint) {
    const res = await fetch(endpoint);
    return res.json();
  }

  async function apiPost(endpoint, body = {}) {
    const res = await fetch(endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    });
    return res.json();
  }

  // Load status
  async function loadStatus() {
    try {
      const data = await apiGet('/api/status');
      state.current = data.current;
      state.palette = data.palette;

      updateCurrentView();
      renderSwatches();
      renderColorInspector();
      renderCodePreview();
    } catch (err) {
      console.error('Failed to load status:', err);
    }
  }

  // Load wallpapers
  async function loadWallpapers() {
    try {
      const data = await apiGet('/api/wallpapers');
      state.wallpapers = data.wallpapers || [];
      wallpaperCount.textContent = state.wallpapers.length;

      renderTopicFilters();
      renderWallpaperGrid();
    } catch (err) {
      console.error('Failed to load wallpapers:', err);
    }
  }

  // Load config
  async function loadConfig() {
    try {
      const data = await apiGet('/api/config');
      state.config = data;

      document.getElementById('cfgTopics').value = (data.topics?.list || []).join(', ');
      document.getElementById('cfgCount').value = data.topics?.count_per_topic || 4;
      document.getElementById('srcWallhaven').checked = !!data.sources?.wallhaven?.enabled;
      document.getElementById('srcBing').checked = !!data.sources?.bing?.enabled;
      document.getElementById('srcNASA').checked = !!data.sources?.nasa?.enabled;
      document.getElementById('srcReddit').checked = !!data.sources?.reddit?.enabled;

      document.getElementById('cfgRotateInterval').value = data.general?.rotation_interval || '1h';
      document.getElementById('cfgMaxCached').value = data.general?.max_cached || 60;
      document.getElementById('cfgSetter').value = data.general?.de_setter || 'auto';
    } catch (err) {
      console.error('Failed to load config:', err);
    }
  }

  // Update current view
  function updateCurrentView() {
    if (!state.current) return;

    currentTitle.textContent = state.current.title || state.current.id;
    currentTopic.textContent = `${state.current.topic} (${state.current.source})`;
    currentDim.textContent = `${state.current.width}x${state.current.height}`;
    currentThumb.src = `/api/wallpaper/image?id=${encodeURIComponent(state.current.id)}`;

    if (state.palette) {
      currentMode.textContent = state.palette.is_dark ? 'Dark' : 'Light';
      document.documentElement.style.setProperty('--accent', state.palette.accent);
      document.documentElement.style.setProperty('--accent-glow', `${state.palette.accent}40`);
      document.documentElement.style.setProperty('--accent-2', state.palette.accent_secondary);
      document.documentElement.style.setProperty('--border-active', state.palette.accent);
    }
  }

  // Render swatches bar
  function renderSwatches() {
    if (!state.palette) return;
    const p = state.palette;

    const items = [
      { name: 'Background', hex: p.background },
      { name: 'Surface', hex: p.surface },
      { name: 'Accent', hex: p.accent },
      { name: 'Accent 2', hex: p.accent_secondary },
      { name: 'Border', hex: p.active_border },
      { name: 'Foreground', hex: p.foreground }
    ];

    swatchesContainer.innerHTML = '';
    items.forEach(it => {
      const sw = document.createElement('div');
      sw.className = 'swatch';
      sw.style.backgroundColor = it.hex;
      sw.title = `${it.name}: ${it.hex} (Click to copy)`;
      sw.addEventListener('click', () => copyToClipboard(it.hex, it.name));
      swatchesContainer.appendChild(sw);
    });

    ansiContainer.innerHTML = '';
    if (p.colors && p.colors.length) {
      p.colors.forEach((col, idx) => {
        const chip = document.createElement('div');
        chip.className = 'ansi-chip';
        chip.style.backgroundColor = col;
        chip.title = `Color${idx}: ${col} (Click to copy)`;
        chip.addEventListener('click', () => copyToClipboard(col, `ANSI ${idx}`));
        ansiContainer.appendChild(chip);
      });
    }
  }

  // Render topic filter pills
  function renderTopicFilters() {
    const topics = new Set();
    state.wallpapers.forEach(w => {
      if (w.topic) topics.add(w.topic);
    });

    topicFilters.innerHTML = '';
    const allBtn = document.createElement('button');
    allBtn.className = `filter-pill ${state.activeTopicFilter === '' ? 'active' : ''}`;
    allBtn.textContent = 'All';
    allBtn.addEventListener('click', () => {
      state.activeTopicFilter = '';
      renderTopicFilters();
      renderWallpaperGrid();
    });
    topicFilters.appendChild(allBtn);

    topics.forEach(top => {
      const btn = document.createElement('button');
      btn.className = `filter-pill ${state.activeTopicFilter === top ? 'active' : ''}`;
      btn.textContent = top;
      btn.addEventListener('click', () => {
        state.activeTopicFilter = top;
        renderTopicFilters();
        renderWallpaperGrid();
      });
      topicFilters.appendChild(btn);
    });
  }

  // Render wallpaper grid
  function renderWallpaperGrid() {
    wallpaperGrid.innerHTML = '';

    const filtered = state.wallpapers.filter(w => {
      if (state.activeTopicFilter && w.topic !== state.activeTopicFilter) return false;
      if (state.searchQuery) {
        const q = state.searchQuery;
        const matchTitle = (w.title || '').toLowerCase().includes(q);
        const matchTopic = (w.topic || '').toLowerCase().includes(q);
        const matchSource = (w.source || '').toLowerCase().includes(q);
        if (!matchTitle && !matchTopic && !matchSource) return false;
      }
      return true;
    });

    if (filtered.length === 0) {
      wallpaperGrid.innerHTML = '<div class="empty-msg">No wallpapers found matching the filter.</div>';
      return;
    }

    filtered.forEach(w => {
      const isCurrent = state.current && state.current.id === w.id;
      const card = document.createElement('div');
      card.className = `wallpaper-card ${isCurrent ? 'current' : ''}`;

      card.innerHTML = `
        <div class="card-media">
          <img src="/api/wallpaper/image?id=${encodeURIComponent(w.id)}" alt="${w.title}" loading="lazy">
          <div class="card-badges">
            <span class="card-badge">${w.topic}</span>
            <span class="card-badge">${w.source}</span>
          </div>
          <button class="btn-star ${w.favorite ? 'favorited' : ''}" title="Favorite">★</button>
        </div>
        <div class="card-body">
          <div class="card-title" title="${w.title}">${w.title}</div>
          <div class="card-meta">
            <span>${w.width}x${w.height}</span>
            <span>${(w.size_bytes / (1024 * 1024)).toFixed(1)} MB</span>
          </div>
          <div class="card-actions">
            <button class="btn btn-primary btn-apply">${isCurrent ? 'Active' : 'Apply'}</button>
            <button class="btn btn-secondary btn-del" title="Blacklist and remove">✕</button>
          </div>
        </div>
      `;

      // Apply button
      card.querySelector('.btn-apply').addEventListener('click', async () => {
        showToast('Applying wallpaper & theme...');
        const res = await apiPost('/api/set', { id: w.id });
        if (res.success) {
          showToast('Wallpaper & theme applied!');
          loadStatus();
          loadWallpapers();
        } else {
          showToast(`Error: ${res.error}`);
        }
      });

      // Favorite button
      card.querySelector('.btn-star').addEventListener('click', async (e) => {
        e.stopPropagation();
        const res = await apiPost('/api/favorite', { id: w.id });
        if (res.success) {
          w.favorite = res.favorite;
          e.target.classList.toggle('favorited', res.favorite);
          showToast(res.favorite ? 'Pinned as favorite!' : 'Removed from favorites');
        }
      });

      // Blacklist button
      card.querySelector('.btn-del').addEventListener('click', async () => {
        const res = await apiPost('/api/blacklist', { id: w.id });
        if (res.success) {
          showToast('Wallpaper blacklisted and removed');
          loadStatus();
          loadWallpapers();
        }
      });

      wallpaperGrid.appendChild(card);
    });
  }

  // Render color inspector
  function renderColorInspector() {
    if (!state.palette) return;
    const p = state.palette;
    colorCardsContainer.innerHTML = '';

    const colors = [
      { name: 'Background', hex: p.background, desc: 'Base dark/light canvas' },
      { name: 'Surface', hex: p.surface, desc: 'Card/modal surface' },
      { name: 'Primary Accent', hex: p.accent, desc: 'Vibrant highlight' },
      { name: 'Secondary Accent', hex: p.accent_secondary, desc: 'Harmonic complement' },
      { name: 'Active Border', hex: p.active_border, desc: 'Hyprland border' },
      { name: 'Foreground', hex: p.foreground, desc: 'High contrast text' }
    ];

    colors.forEach(c => {
      const card = document.createElement('div');
      card.className = 'color-card';
      card.innerHTML = `
        <div class="color-preview" style="background-color: ${c.hex};"></div>
        <div class="color-info">
          <span class="color-name">${c.name}</span>
          <span class="color-hex">${c.hex}</span>
        </div>
      `;
      card.addEventListener('click', () => copyToClipboard(c.hex, c.name));
      colorCardsContainer.appendChild(card);
    });
  }

  // Render code preview
  function renderCodePreview() {
    if (!state.palette) return;
    const p = state.palette;

    let code = '';
    switch (state.activeExportTab) {
      case 'hyprland':
        code = `# Generated by syncpaper
$background = ${toHypr(p.background)}
$foreground = ${toHypr(p.foreground)}
$surface = ${toHypr(p.surface)}
$accent = ${toHypr(p.accent)}
$accent_secondary = ${toHypr(p.accent_secondary)}
$active_border_1 = ${toHypr(p.active_border)}
$active_border_2 = ${toHypr(p.accent_secondary)}
$inactive_border = ${toHypr(p.inactive_border)}`;
        break;
      case 'waybar':
        code = `/* Generated by syncpaper */
@define-color background ${p.background};
@define-color foreground ${p.foreground};
@define-color surface ${p.surface};
@define-color accent ${p.accent};
@define-color accent_secondary ${p.accent_secondary};
@define-color active_border ${p.active_border};
@define-color inactive_border ${p.inactive_border};`;
        break;
      case 'kitty':
        code = `# Generated by syncpaper
background ${p.background}
foreground ${p.foreground}
cursor ${p.cursor}
selection_background ${p.accent}
selection_foreground ${p.background}
active_border_color ${p.active_border}
inactive_border_color ${p.inactive_border}
${(p.colors || []).map((col, idx) => `color${idx} ${col}`).join('\n')}`;
        break;
      case 'json':
        code = JSON.stringify({
          wallpaper: state.current ? state.current.local_path : '',
          alpha: "100",
          special: {
            background: p.background,
            foreground: p.foreground,
            cursor: p.cursor,
            accent: p.accent
          },
          colors: (p.colors || []).reduce((acc, col, idx) => {
            acc[`color${idx}`] = col;
            return acc;
          }, {})
        }, null, 2);
        break;
    }

    codeBlock.textContent = code;
  }

  function toHypr(hex) {
    return `rgb(${hex.replace('#', '')})`;
  }

  // Rotate button
  btnRotate.addEventListener('click', async () => {
    btnRotate.disabled = true;
    showToast('Rotating wallpaper and theme...');
    try {
      const res = await apiPost('/api/rotate');
      if (res.success) {
        showToast('New wallpaper & theme applied!');
        await loadStatus();
        await loadWallpapers();
      } else {
        showToast(`Rotation error: ${res.error}`);
      }
    } catch (e) {
      showToast('Rotation failed');
    } finally {
      btnRotate.disabled = false;
    }
  });

  // Sync button
  btnSync.addEventListener('click', async () => {
    btnSync.disabled = true;
    showToast('Fetching fresh wallpapers online...');
    try {
      const res = await apiPost('/api/sync');
      if (res.success) {
        showToast(`Sync complete! Downloaded ${res.downloaded} new wallpapers.`);
        await loadWallpapers();
      } else {
        showToast(`Sync warning: ${res.error}`);
      }
    } catch (e) {
      showToast('Sync request failed');
    } finally {
      btnSync.disabled = false;
    }
  });

  // Save config button
  btnSaveConfig.addEventListener('click', async () => {
    const rawTopics = document.getElementById('cfgTopics').value;
    const topicsList = rawTopics.split(',').map(s => s.trim()).filter(Boolean);

    const payload = {
      topics: {
        list: topicsList,
        count_per_topic: parseInt(document.getElementById('cfgCount').value, 10) || 4
      },
      sources: {
        wallhaven: { enabled: document.getElementById('srcWallhaven').checked },
        bing: { enabled: document.getElementById('srcBing').checked },
        nasa: { enabled: document.getElementById('srcNASA').checked },
        reddit: { enabled: document.getElementById('srcReddit').checked }
      },
      general: {
        rotation_interval: document.getElementById('cfgRotateInterval').value || '1h',
        max_cached: parseInt(document.getElementById('cfgMaxCached').value, 10) || 60,
        de_setter: document.getElementById('cfgSetter').value || 'auto'
      }
    };

    try {
      const res = await apiPost('/api/config', payload);
      if (res.success) {
        saveStatus.textContent = 'Saved successfully!';
        setTimeout(() => saveStatus.textContent = '', 2500);
        showToast('Configuration updated!');
      } else {
        saveStatus.textContent = 'Error saving';
      }
    } catch (e) {
      saveStatus.textContent = 'Error saving';
    }
  });

  // Send shutdown signal when browser window is closed
  window.addEventListener('beforeunload', () => {
    navigator.sendBeacon('/api/close');
  });

  // Initial load
  loadStatus();
  loadWallpapers();
  loadConfig();
});
