// PromisedLand landing — theme switcher, interactive preview, spoiler curtain, interest graph & RSVP.
(function () {
  'use strict'

  var reduceMotion = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches

  // ── Theme toggle (Dark default + Warm Washi Light) ───────────────────────
  var root = document.documentElement
  var metaTheme = document.getElementById('meta-theme-color')
  var themeButtons = [document.getElementById('theme-toggle'), document.getElementById('theme-toggle-mobile')].filter(Boolean)

  function applyTheme(theme) {
    root.setAttribute('data-theme', theme)
    if (metaTheme) metaTheme.setAttribute('content', theme === 'light' ? '#f9f7f3' : '#090a0d')
    themeButtons.forEach(function (btn) {
      btn.setAttribute('aria-label', theme === 'light' ? 'Switch to dark theme' : 'Switch to light theme')
    })
    try { localStorage.setItem('pl-theme', theme) } catch (e) {}
  }

  themeButtons.forEach(function (btn) {
    btn.addEventListener('click', function () {
      var current = root.getAttribute('data-theme') === 'light' ? 'light' : 'dark'
      applyTheme(current === 'light' ? 'dark' : 'light')
    })
  })

  // ── Mobile navigation ────────────────────────────────────────────────────
  var toggle = document.getElementById('nav-toggle')
  var nav = document.getElementById('site-nav')
  if (toggle && nav) {
    toggle.addEventListener('click', function () {
      var open = toggle.getAttribute('aria-expanded') === 'true'
      toggle.setAttribute('aria-expanded', String(!open))
      toggle.setAttribute('aria-label', open ? 'Open navigation' : 'Close navigation')
      nav.classList.toggle('is-open', !open)
    })
    nav.addEventListener('click', function (e) {
      if (e.target.closest('a')) {
        toggle.setAttribute('aria-expanded', 'false')
        nav.classList.remove('is-open')
      }
    })
  }

  // ── Hero preview feed tabs ───────────────────────────────────────────────
  var previewTabs = [].slice.call(document.querySelectorAll('[data-preview-tab]'))
  previewTabs.forEach(function (tab) {
    tab.addEventListener('click', function () {
      previewTabs.forEach(function (t) { t.classList.remove('is-active') })
      tab.classList.add('is-active')
    })
  })

  // ── Scroll reveals ───────────────────────────────────────────────────────
  var reveals = [].slice.call(document.querySelectorAll('.reveal'))
  if (reduceMotion || !('IntersectionObserver' in window)) {
    reveals.forEach(function (el) { el.classList.add('is-visible') })
  } else {
    var observer = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry) {
        if (!entry.isIntersecting) return
        var siblings = [].slice.call(entry.target.parentNode.children).filter(function (c) { return c.classList.contains('reveal') })
        entry.target.style.transitionDelay = (Math.max(0, siblings.indexOf(entry.target)) * 40) + 'ms'
        entry.target.classList.add('is-visible')
        observer.unobserve(entry.target)
      })
    }, { threshold: 0.12, rootMargin: '0px 0px -32px 0px' })
    reveals.forEach(function (el) { observer.observe(el) })
  }

  // ── Spoiler curtain demo ─────────────────────────────────────────────────
  var spoiler = document.getElementById('spoiler-demo')
  var spoilerToggle = document.getElementById('spoiler-toggle')
  if (spoiler && spoilerToggle) {
    spoilerToggle.addEventListener('click', function () {
      var hidden = spoiler.getAttribute('data-hidden') === 'true'
      spoiler.setAttribute('data-hidden', String(!hidden))
      spoilerToggle.setAttribute('aria-expanded', String(hidden))
      spoilerToggle.textContent = hidden ? 'Hide spoiler' : 'Reveal spoiler'
    })
  }

  // ── Event RSVP demo ──────────────────────────────────────────────────────
  var rsvpBtn = document.getElementById('rsvp-toggle')
  var rsvpCount = document.getElementById('rsvp-count')
  if (rsvpBtn && rsvpCount) {
    rsvpBtn.addEventListener('click', function () {
      var going = rsvpBtn.getAttribute('aria-pressed') === 'true'
      rsvpBtn.setAttribute('aria-pressed', String(!going))
      rsvpBtn.classList.toggle('btn--primary', !going)
      rsvpBtn.classList.toggle('btn--secondary', going)
      rsvpBtn.textContent = going ? 'RSVP to screening' : 'Going · Cancel RSVP'
      rsvpCount.textContent = going ? '18 / 24 going' : '19 / 24 going'
    })
  }

  // ── Interactive interest graph ───────────────────────────────────────────
  var interests = [
    { id: 'frieren', label: 'Frieren', x: 96, y: 78 },
    { id: 'chainsaw', label: 'Chainsaw Man', x: 248, y: 72 },
    { id: 'jjk', label: 'Jujutsu Kaisen', x: 408, y: 104 },
    { id: 'manga', label: 'Manga', x: 128, y: 220 },
    { id: 'cosplay', label: 'Cosplay', x: 284, y: 186 },
    { id: 'fanart', label: 'Fan art', x: 452, y: 232 },
    { id: 'conventions', label: 'Conventions', x: 200, y: 280 }
  ]

  var chips = document.getElementById('graph-chips')
  var svg = document.getElementById('graph-svg')
  var readout = document.getElementById('graph-readout')
  if (!chips || !svg || !readout) return

  var NS = 'http://www.w3.org/2000/svg'
  var selected = { frieren: true, cosplay: true }

  function el(name, attrs) {
    var node = document.createElementNS(NS, name)
    Object.keys(attrs || {}).forEach(function (k) { node.setAttribute(k, attrs[k]) })
    return node
  }

  interests.forEach(function (interest) {
    var chip = document.createElement('button')
    chip.type = 'button'
    chip.className = 'chip'
    chip.textContent = interest.label
    chip.setAttribute('aria-pressed', String(!!selected[interest.id]))
    chip.dataset.id = interest.id
    chip.addEventListener('click', function () {
      selected[interest.id] = !selected[interest.id]
      chip.setAttribute('aria-pressed', String(!!selected[interest.id]))
      render()
    })
    chips.appendChild(chip)
  })

  function render() {
    while (svg.firstChild) svg.removeChild(svg.firstChild)

    var grid = el('g', { stroke: 'var(--grid-line)', 'stroke-width': '1' })
    for (var x = 40; x < 560; x += 52) grid.appendChild(el('line', { x1: x, y1: 0, x2: x, y2: 320 }))
    for (var y = 40; y < 320; y += 52) grid.appendChild(el('line', { x1: 0, y1: y, x2: 560, y2: y }))
    svg.appendChild(grid)

    var picked = interests.filter(function (i) { return selected[i.id] })
    var lines = el('g', { 'aria-hidden': 'true' })
    for (var a = 0; a < picked.length; a++) {
      for (var b = a + 1; b < picked.length; b++) {
        lines.appendChild(el('line', {
          x1: picked[a].x, y1: picked[a].y, x2: picked[b].x, y2: picked[b].y,
          class: 'graph__line'
        }))
      }
    }
    svg.appendChild(lines)

    picked.forEach(function (i, index) {
      var group = el('g', { class: 'graph__star' })
      group.appendChild(el('circle', { cx: i.x, cy: i.y, r: index === 0 ? 6 : 4.5, fill: 'var(--accent)' }))
      group.appendChild(el('circle', { cx: i.x, cy: i.y, r: 12, fill: 'none', stroke: 'var(--accent)', 'stroke-opacity': '0.35' }))
      var text = el('text', { x: i.x + 14, y: i.y + 4, class: 'graph__label' })
      text.textContent = i.label
      group.appendChild(text)
      svg.appendChild(group)
    })

    if (!picked.length) {
      var hint = el('text', { x: 280, y: 164, 'text-anchor': 'middle', fill: 'var(--text-muted)', 'font-size': '13', 'font-family': 'var(--font-mono)' })
      hint.textContent = 'Select interests on the left to map your graph'
      svg.appendChild(hint)
    }

    var count = picked.length
    var links = (count * (count - 1)) / 2
    readout.textContent = count === 0
      ? 'Choose an interest to plot it'
      : count + ' interest' + (count === 1 ? '' : 's') + ' plotted · ' + links + ' connection' + (links === 1 ? '' : 's')
  }

  render()
})()
