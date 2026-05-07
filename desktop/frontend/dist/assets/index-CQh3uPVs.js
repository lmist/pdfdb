(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const a of document.querySelectorAll('link[rel="modulepreload"]'))s(a);new MutationObserver(a=>{for(const r of a)if(r.type==="childList")for(const y of r.addedNodes)y.tagName==="LINK"&&y.rel==="modulepreload"&&s(y)}).observe(document,{childList:!0,subtree:!0});function n(a){const r={};return a.integrity&&(r.integrity=a.integrity),a.referrerPolicy&&(r.referrerPolicy=a.referrerPolicy),a.crossOrigin==="use-credentials"?r.credentials="include":a.crossOrigin==="anonymous"?r.credentials="omit":r.credentials="same-origin",r}function s(a){if(a.ep)return;a.ep=!0;const r=n(a);fetch(a.href,r)}})();const C=window.go?.main?.App,S=document.querySelector("#app");let o=D(),b="",m="",c="",p=!1,$="default",h="",v="";if(!S)throw new Error("missing app root");const x=S;function D(){return{profiles:[],documents:[],health:{ready:!1,cached:0,total:0,cacheDir:"",message:"loading"},needsDb:!0}}function i(e,...t){const n=C?.[e];return n?n(...t):Promise.reject(new Error(`desktop bridge is not ready: ${e}`))}async function w(e=!1){try{e||(c="Refreshing");const t=await i("GetState"),n=JSON.stringify(t)!==JSON.stringify(o);o=t,o.needsDb&&(p=!0),e||(c=""),(n||!e)&&f()}catch(t){o.error=k(t),e||(c=""),f()}}async function l(e,t){m=e,c=e,f();try{await t(),c="Done",await w(!0)}catch(n){c=k(n)}finally{m="",f()}}function E(){const e=b.trim().toLowerCase();return e?o.documents.filter(t=>[t.title,t.slug,t.filename,t.sourceUrl??""].join(" ").toLowerCase().includes(e)):o.documents}const P='<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33h.01a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82v.01a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>',L='<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>',O='<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="7"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>';function f(){const e=E(),t=o.profiles.find(s=>s.active),n=c||`${o.documents.length} ${o.documents.length===1?"document":"documents"}`;x.innerHTML=`
    <section class="shell">
      <header class="titlebar">
        <div class="brand">
          <strong>PDF DB</strong>
          <span class="profile-name">${u(t?.name??"No database")}</span>
        </div>
        <button class="icon" data-action="settings" title="Database settings">${P}</button>
      </header>

      <section class="health ${o.health.ready?"ok":"warn"}">
        <span class="dot"></span>
        <p>${u(o.health.message||"Checking")}</p>
        <button data-action="warm" ${g()}>Warm cache</button>
      </section>

      <section class="search">
        ${O}
        <input id="search" type="search" placeholder="Search documents" value="${d(b)}" autocomplete="off" spellcheck="false" />
      </section>

      <section class="list" aria-label="PDFs">
        ${e.map(q).join("")||'<p class="empty">No documents yet.<br/>Paste a URL below to import.</p>'}
      </section>

      <section class="ingest">
        <input id="ingest" type="text" placeholder="Paste URL or drop a path" value="${d(v)}" autocomplete="off" spellcheck="false" />
        <button data-action="ingest-url" ${g()}>Import</button>
        <button data-action="ingest-file" ${g()}>Browse</button>
      </section>

      <footer class="${m?"busy":""}">
        <span class="pulse"></span>
        <span>${u(n)}</span>
      </footer>
    </section>
    ${p?N():""}
  `,I()}function q(e){const t=e.pageCount?`${e.pageCount} pages`:e.filename;return`
    <article class="row ${e.open?"open":""}" data-slug="${d(e.slug)}" title="Double-click to open">
      <div class="glyph">PDF</div>
      <div class="meta">
        <h2>${u(e.title||e.filename||e.slug)}</h2>
        <p>${u(t)} &middot; ${A(e.sizeBytes)}</p>
      </div>
      <div class="actions">
        ${e.open?`<button class="icon close-btn" data-action="close" data-slug="${d(e.slug)}" title="Close in Zathura">${L}</button>`:""}
      </div>
    </article>
  `}function N(){return`
    <aside class="modal">
      <section class="panel">
        <header>
          <strong>Database</strong>
          <button class="icon" data-action="settings" title="Close">${L}</button>
        </header>
        <label>
          <span>Profile</span>
          <input id="profile-name" value="${d($)}" autocomplete="off" spellcheck="false" />
        </label>
        <label>
          <span>Postgres URL</span>
          <input id="database-url" type="password" value="${d(h)}" placeholder="postgresql://..." autocomplete="off" spellcheck="false" />
        </label>
        <button class="primary" data-action="save-profile" ${g()}>Save to Keychain</button>
        ${o.profiles.length?`
          <div class="profiles">
            ${o.profiles.map(e=>`
              <button data-action="switch-profile" data-name="${d(e.name)}" class="${e.active?"active":""}">
                ${u(e.name)}
              </button>
            `).join("")}
          </div>
        `:""}
      </section>
    </aside>
  `}function I(){document.querySelector("#search")?.addEventListener("input",e=>{b=e.target.value,f()}),document.querySelector("#ingest")?.addEventListener("input",e=>{v=e.target.value}),document.querySelector("#profile-name")?.addEventListener("input",e=>{$=e.target.value}),document.querySelector("#database-url")?.addEventListener("input",e=>{h=e.target.value}),document.querySelectorAll("[data-action]").forEach(e=>{e.addEventListener("click",B)}),document.querySelectorAll(".row").forEach(e=>{e.addEventListener("click",t=>{if(t.target.closest("[data-action]"))return;const n=e.dataset.slug;n&&l("Opening",()=>i("OpenDocument",n))})})}function B(e){const t=e.currentTarget,n=t.dataset.action;if(n==="settings"){p=!p,f();return}if(n==="warm"){l("Warming cache",()=>i("WarmCache"));return}if(n==="save-profile"){l("Saving profile",async()=>{await i("SaveProfile",$,h),h="",p=!1});return}if(n==="switch-profile"){const s=t.dataset.name;s&&l("Switching database",()=>i("SetActiveProfile",s));return}if(n==="close"){const s=t.dataset.slug;s&&l("Closing",()=>i("CloseDocument",s));return}if(n==="ingest-url"){const s=v;l("Importing",async()=>{await i("IngestSource",s),v=""});return}n==="ingest-file"&&l("Importing file",()=>i("PickAndIngestFile"))}function g(){return m?"disabled":""}function k(e){return e instanceof Error?e.message:String(e)}function A(e){return e?e<1024*1024?`${Math.round(e/1024)} KB`:`${(e/1024/1024).toFixed(1)} MB`:"0 B"}function u(e){return e.replace(/[&<>"']/g,t=>j(t))}function d(e){return u(e)}function j(e){return{"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[e]??e}w();setInterval(()=>{w(!0)},1e3);
