import { readFileSync, writeFileSync } from 'node:fs';
const isomorph=process.env.ISOMORPH_ROOT ?? 'D:/dev/isomorph';
const {parse}=await import(`${isomorph}/src/parser/index.ts`);
const {analyze}=await import(`${isomorph}/src/semantics/analyzer.ts`);
const {renderDiagram}=await import(`${isomorph}/src/renderer/index.ts`);
const colors = {'--iso-text':'#172521','--iso-text-body':'#172521','--iso-text-primary':'#172521','--iso-text-secondary':'#33423d','--iso-text-muted':'#52675f','--iso-bg-panel':'#ffffff','--iso-bg-primary':'#ffffff','--iso-bg-secondary':'#f0f6f2','--iso-border':'#aabcb2','--iso-accent':'#1b7554'};
for (const name of ['strategy','command','observer','sequence']) {
 const base = `${import.meta.dir}/${name}`;
 const parsed=parse(readFileSync(base+'.isx','utf8'));
 if(parsed.errors.length) throw new Error(JSON.stringify(parsed.errors));
 const checked=analyze(parsed.program);
 if(checked.errors.length) throw new Error(JSON.stringify(checked.errors));
 let svg=renderDiagram(checked.iom.diagrams[0]);
 svg=svg.replace(/var\((--[a-z-]+)(?:,[^)]*)?\)/g,(_,key)=>colors[key]??'#edf3ef');
 svg=svg.replaceAll('background:transparent','background:#ffffff');
 svg=svg.replace(/<(?:line|path)[^>]*stroke="transparent"[^>]*\/>/g,'');
 svg=svg.replace(/ filter="url\(#shadow\)"/g,'');
 svg=svg.replace(/fill="url\(#grad-[a-z]+\)"/g,'fill="#edf3ef"');
 svg=svg.replace(/font-family="[^"]*"/g,'font-family="Arial, DejaVu Sans, sans-serif"');
 svg=svg.replace(/(<svg[^>]*>)/,'$1\n<rect width="100%" height="100%" fill="#ffffff"/>');
 writeFileSync(base+'.svg',svg);
 console.log(`PASS ${name}: Isomorph parsing, semantics, SVG export`);
}
