(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const a of document.querySelectorAll('link[rel="modulepreload"]'))i(a);new MutationObserver(a=>{for(const r of a)if(r.type==="childList")for(const v of r.addedNodes)v.tagName==="LINK"&&v.rel==="modulepreload"&&i(v)}).observe(document,{childList:!0,subtree:!0});function n(a){const r={};return a.integrity&&(r.integrity=a.integrity),a.referrerPolicy&&(r.referrerPolicy=a.referrerPolicy),a.crossOrigin==="use-credentials"?r.credentials="include":a.crossOrigin==="anonymous"?r.credentials="omit":r.credentials="same-origin",r}function i(a){if(a.ep)return;a.ep=!0;const r=n(a);fetch(a.href,r)}})();const D=window.go?.main?.App,S=document.querySelector("#app");let s=q(),y="",b="",l="",f=!1,$="default",m="",h="";if(!S)throw new Error("missing app root");const P=S;function q(){return{profiles:[],documents:[],health:{ready:!1,cached:0,total:0,cacheDir:"",message:"loading"},needsDb:!0}}function o(e,...t){const n=D?.[e];return n?n(...t):Promise.reject(new Error(`desktop bridge is not ready: ${e}`))}async function w(e=!1){try{e||(l="Refreshing"),s=await o("GetState"),s.needsDb&&(f=!0),e||(l="")}catch(t){s.error=L(t),e||(l="")}p()}async function c(e,t){b=e,l=e,p();try{await t(),l="Done",await w(!0)}catch(n){l=L(n)}finally{b="",p()}}function E(){const e=y.trim().toLowerCase();return e?s.documents.filter(t=>[t.title,t.slug,t.filename,t.sourceUrl??""].join(" ").toLowerCase().includes(e)):s.documents}function p(){const e=E(),t=s.profiles.find(n=>n.active);P.innerHTML=`
    <section class="shell">
      <header class="titlebar">
        <div>
          <strong>PDF DB</strong>
          <span>${u(t?.name??"No database")}</span>
        </div>
        <button class="icon" data-action="settings" title="Database settings">...</button>
      </header>

      <section class="health ${s.health.ready?"ok":"warn"}">
        <span></span>
        <p>${u(s.health.message||"Checking")}</p>
        <button data-action="warm" ${g()}>Warm</button>
      </section>

      <section class="search">
        <input id="search" type="search" placeholder="Search PDFs" value="${d(y)}" />
      </section>

      <section class="list" aria-label="PDFs">
        ${e.map(C).join("")||'<p class="empty">No PDFs in this database.</p>'}
      </section>

      <section class="ingest">
        <input id="ingest" type="text" placeholder="Paste URL or path" value="${d(h)}" />
        <button data-action="ingest-url" ${g()}>Import</button>
        <button data-action="ingest-file" ${g()}>File</button>
      </section>

      <footer>
        <span>${u(l||`${s.documents.length} PDFs`)}</span>
      </footer>
    </section>
    ${f?F():""}
  `,A()}function C(e){return`
    <article class="row ${e.open?"open":""}" data-slug="${d(e.slug)}" title="Double-click to open">
      <div class="mark"></div>
      <div class="meta">
        <h2>${u(e.title||e.filename||e.slug)}</h2>
        <p>${u(e.pageCount?`${e.pageCount} pages`:e.filename)} · ${I(e.sizeBytes)}</p>
      </div>
      <button data-action="close" data-slug="${d(e.slug)}" ${e.open?"":"disabled"} title="Close in Zathura">Close</button>
    </article>
  `}function F(){return`
    <aside class="modal">
      <section class="panel">
        <header>
          <strong>Database</strong>
          <button class="icon" data-action="settings" title="Close">x</button>
        </header>
        <label>
          <span>Profile</span>
          <input id="profile-name" value="${d($)}" />
        </label>
        <label>
          <span>Postgres URL</span>
          <input id="database-url" type="password" value="${d(m)}" placeholder="postgresql://..." />
        </label>
        <button class="primary" data-action="save-profile" ${g()}>Save to Keychain</button>
        <div class="profiles">
          ${s.profiles.map(e=>`
            <button data-action="switch-profile" data-name="${d(e.name)}" class="${e.active?"active":""}">
              ${u(e.name)}
            </button>
          `).join("")}
        </div>
      </section>
    </aside>
  `}function A(){document.querySelector("#search")?.addEventListener("input",e=>{y=e.target.value,p()}),document.querySelector("#ingest")?.addEventListener("input",e=>{h=e.target.value}),document.querySelector("#profile-name")?.addEventListener("input",e=>{$=e.target.value}),document.querySelector("#database-url")?.addEventListener("input",e=>{m=e.target.value}),document.querySelectorAll("[data-action]").forEach(e=>{e.addEventListener("click",O)}),document.querySelectorAll(".row").forEach(e=>{e.addEventListener("dblclick",()=>{const t=e.dataset.slug;t&&c("Opening",()=>o("OpenDocument",t))})})}function O(e){const t=e.currentTarget,n=t.dataset.action;if(n==="settings"){f=!f,p();return}if(n==="warm"){c("Warming cache",()=>o("WarmCache"));return}if(n==="save-profile"){c("Saving profile",async()=>{await o("SaveProfile",$,m),m="",f=!1});return}if(n==="switch-profile"){const i=t.dataset.name;i&&c("Switching database",()=>o("SetActiveProfile",i));return}if(n==="close"){const i=t.dataset.slug;i&&c("Closing",()=>o("CloseDocument",i));return}if(n==="ingest-url"){const i=h;c("Importing",async()=>{await o("IngestSource",i),h=""});return}n==="ingest-file"&&c("Importing file",()=>o("PickAndIngestFile"))}function g(){return b?"disabled":""}function L(e){return e instanceof Error?e.message:String(e)}function I(e){return e?e<1024*1024?`${Math.round(e/1024)} KB`:`${(e/1024/1024).toFixed(1)} MB`:"0 B"}function u(e){return e.replace(/[&<>"']/g,t=>B(t))}function d(e){return u(e)}function B(e){return{"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[e]??e}w();setInterval(()=>{w(!0)},1e3);
