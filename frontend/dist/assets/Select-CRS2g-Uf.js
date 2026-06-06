import{c as i,b as t,r as m,j as s}from"./index-DjNJeCI_.js";/**
 * @license lucide-react v0.400.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const x=i("ChevronDown",[["path",{d:"m6 9 6 6 6-6",key:"qrunsl"}]]);/**
 * @license lucide-react v0.400.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const v=i("Pencil",[["path",{d:"M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z",key:"1a8usu"}],["path",{d:"m15 5 4 4",key:"1mk7zo"}]]);/**
 * @license lucide-react v0.400.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const b=i("Plus",[["path",{d:"M5 12h14",key:"1ays0h"}],["path",{d:"M12 5v14",key:"s699le"}]]),g={list:e=>t.get("/companies",{params:e}),getById:e=>t.get(`/companies/${e}`),create:e=>t.post("/companies",e),update:(e,l)=>t.put(`/companies/${e}`,l),delete:e=>t.delete(`/companies/${e}`)},h=m.forwardRef(({label:e,options:l,error:a,placeholder:n,id:d,className:c="",...p},u)=>{const o=d??(e?e.toLowerCase().replace(/\s+/g,"-"):void 0);return s.jsxs("div",{className:"flex flex-col gap-1",children:[e&&s.jsx("label",{htmlFor:o,className:"text-sm font-medium text-slate-700",children:e}),s.jsxs("div",{className:"relative",children:[s.jsxs("select",{ref:u,id:o,"aria-describedby":a?`${o}-error`:void 0,"aria-invalid":a?"true":void 0,className:["w-full appearance-none px-3 py-2 pr-9 text-sm rounded-lg border bg-white text-slate-800","focus:outline-none focus:ring-2 focus:ring-offset-0","disabled:bg-slate-50 disabled:text-slate-400 disabled:cursor-not-allowed","transition-colors",a?"border-red-400 focus:ring-red-400 focus:border-red-400":"border-slate-300 focus:ring-sky-500 focus:border-sky-500",c].filter(Boolean).join(" "),...p,children:[n&&s.jsx("option",{value:"",disabled:!0,children:n}),l.map(r=>s.jsx("option",{value:r.value,children:r.label},r.value))]}),s.jsx(x,{size:16,className:"pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-slate-400","aria-hidden":"true"})]}),a&&s.jsx("p",{id:`${o}-error`,role:"alert",className:"text-xs text-red-600",children:a})]})});h.displayName="Select";export{b as P,h as S,v as a,g as c};
