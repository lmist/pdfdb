(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const a of document.querySelectorAll('link[rel="modulepreload"]'))o(a);new MutationObserver(a=>{for(const r of a)if(r.type==="childList")for(const b of r.addedNodes)b.tagName==="LINK"&&b.rel==="modulepreload"&&o(b)}).observe(document,{childList:!0,subtree:!0});function n(a){const r={};return a.integrity&&(r.integrity=a.integrity),a.referrerPolicy&&(r.referrerPolicy=a.referrerPolicy),a.crossOrigin==="use-credentials"?r.credentials="include":a.crossOrigin==="anonymous"?r.credentials="omit":r.credentials="same-origin",r}function o(a){if(a.ep)return;a.ep=!0;const r=n(a);fetch(a.href,r)}})();function h(e){return e?"disabled":""}function L(e){return e instanceof Error?e.message:String(e)}function D(e){return e?e<1024*1024?`${Math.round(e/1024)} KB`:`${(e/1024/1024).toFixed(1)} MB`:"0 B"}function u(e){return e.replace(/[&<>"']/g,t=>x(t))}function d(e){return u(e)}function x(e){return{"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[e]??e}const P=window.go?.main?.App,k=document.querySelector("#app");let s=O(),w="",l="",f="",p=!1,$="default",v="",y="";if(!k)throw new Error("missing app root");const I=k;function O(){return{profiles:[],documents:[],health:{ready:!1,cached:0,total:0,cacheDir:"",message:"loading"},needsDb:!0}}function i(e,...t){const n=P?.[e];return n?n(...t):Promise.reject(new Error(`desktop bridge is not ready: ${e}`))}async function S(e=!1){try{e||(f="Refreshing");const t=await i("GetState"),n=JSON.stringify(t)!==JSON.stringify(s);s=t,s.needsDb&&(p=!0),e||(f=""),(n||!e)&&g()}catch(t){s.error=L(t),e||(f=""),g()}}async function c(e,t){l=e,f=e,g();try{await t(),f="Done",await S(!0)}catch(n){f=L(n)}finally{l="",g()}}function q(){const e=w.trim().toLowerCase();return e?s.documents.filter(t=>[t.title,t.slug,t.filename,t.sourceUrl??""].join(" ").toLowerCase().includes(e)):s.documents}const N='<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33h.01a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82v.01a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>',E='<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>',B='<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="7"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>';function g(){const e=q(),t=s.profiles.find(o=>o.active),n=f||`${s.documents.length} ${s.documents.length===1?"document":"documents"}`;I.innerHTML=`
    <section class="shell">
      <header class="titlebar">
        <div class="brand">
          <strong>PDF DB</strong>
          <span class="profile-name">${u(t?.name??"No database")}</span>
        </div>
        <button class="icon" data-action="settings" title="Database settings">${N}</button>
      </header>

      <section class="health ${s.health.ready?"ok":"warn"}">
        <span class="dot"></span>
        <p>${u(s.health.message||"Checking")}</p>
        <button data-action="warm" ${h(l)}>Warm cache</button>
      </section>

      <section class="search">
        ${B}
        <input id="search" type="search" placeholder="Search documents" value="${d(w)}" autocomplete="off" spellcheck="false" aria-label="Search documents" />
      </section>

      <section class="list" aria-label="PDFs">
        ${e.map(T).join("")||'<p class="empty">No documents yet.<br/>Paste a URL below to import.</p>'}
      </section>

      <section class="ingest">
        <input id="ingest" type="text" placeholder="Paste URL or drop a path" value="${d(y)}" autocomplete="off" spellcheck="false" aria-label="Import URL or path" />
        <button data-action="ingest-url" ${h(l)}>Import</button>
        <button data-action="ingest-file" ${h(l)}>Browse</button>
      </section>

      <footer class="${l?"busy":""}">
        <span class="pulse"></span>
        <span>${u(n)}</span>
      </footer>
    </section>
    ${p?A():""}
  `,j()}function T(e){const t=e.pageCount?`${e.pageCount} pages`:e.filename;return`
    <article class="row ${e.open?"open":""}" data-slug="${d(e.slug)}" title="Double-click to open">
      <div class="glyph">PDF</div>
      <div class="meta">
        <h2>${u(e.title||e.filename||e.slug)}</h2>
        <p>${u(t)} &middot; ${D(e.sizeBytes)}</p>
      </div>
      <div class="actions">
        ${e.open?`<button class="icon close-btn" data-action="close" data-slug="${d(e.slug)}" title="Close in Zathura">${E}</button>`:""}
      </div>
    </article>
  `}function A(){return`
    <aside class="modal">
      <section class="panel">
        <header>
          <strong>Database</strong>
          <button class="icon" data-action="settings" title="Close">${E}</button>
        </header>
        <label>
          <span>Profile</span>
          <input id="profile-name" value="${d($)}" autocomplete="off" spellcheck="false" />
        </label>
        <label>
          <span>Postgres URL</span>
          <input id="database-url" type="password" value="${d(v)}" placeholder="postgresql://..." autocomplete="off" spellcheck="false" />
        </label>
        <button class="primary" data-action="save-profile" ${h(l)}>Save to Keychain</button>
        ${s.profiles.length?`
          <div class="profiles">
            ${s.profiles.map(e=>`
              <button data-action="switch-profile" data-name="${d(e.name)}" class="${e.active?"active":""}">
                ${u(e.name)}
              </button>
            `).join("")}
          </div>
        `:""}
      </section>
    </aside>
  `}let m;function C(){const e=y;c("Importing",async()=>{await i("IngestSource",e),y=""})}function j(){document.querySelector("#search")?.addEventListener("input",t=>{w=t.target.value,m!==void 0&&clearTimeout(m),m=window.setTimeout(()=>{m=void 0,g()},150)});const e=document.querySelector("#ingest");e?.addEventListener("input",t=>{y=t.target.value}),e?.addEventListener("keydown",t=>{t.key==="Enter"&&!l&&(t.preventDefault(),C())}),document.querySelector("#profile-name")?.addEventListener("input",t=>{$=t.target.value}),document.querySelector("#database-url")?.addEventListener("input",t=>{v=t.target.value}),document.querySelectorAll("[data-action]").forEach(t=>{t.addEventListener("click",R)}),document.querySelectorAll(".row").forEach(t=>{t.addEventListener("click",n=>{if(n.target.closest("[data-action]"))return;const o=t.dataset.slug;o&&c("Opening",()=>i("OpenDocument",o))})})}function R(e){const t=e.currentTarget,n=t.dataset.action;if(n==="settings"){p=!p,g();return}if(n==="warm"){c("Warming cache",()=>i("WarmCache"));return}if(n==="save-profile"){c("Saving profile",async()=>{await i("SaveProfile",$,v),v="",p=!1});return}if(n==="switch-profile"){const o=t.dataset.name;o&&c("Switching database",()=>i("SetActiveProfile",o));return}if(n==="close"){const o=t.dataset.slug;o&&c("Closing",()=>i("CloseDocument",o));return}if(n==="ingest-url"){C();return}n==="ingest-file"&&c("Importing file",()=>i("PickAndIngestFile"))}S();const U=window.setInterval(()=>{S(!0)},1e3);window.addEventListener("beforeunload",()=>{window.clearInterval(U),m!==void 0&&clearTimeout(m)});window.addEventListener("keydown",e=>{e.key==="Escape"&&p&&(p=!1,g())});
