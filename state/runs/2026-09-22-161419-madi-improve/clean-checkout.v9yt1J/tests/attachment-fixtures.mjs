// Local, generated PDF/scan fixture: no third-party documents or network fetch.
import { chromium } from "playwright";
import { readFile, writeFile, mkdir } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),"..");
const output=process.argv[2];if(!output||!path.isAbsolute(output))throw new Error("absolute dedicated fixture directory required");
await mkdir(output,{recursive:true,mode:0o700});
const fontRoot=path.join(root,"web/node_modules/@fontsource-variable/noto-sans-kr");
let css=await readFile(path.join(fontRoot,"index.css"),"utf8");
for(const match of [...css.matchAll(/url\(([^)]+)\)/g)]){
 const name=match[1].replace(/["']/g,"");
 if(!name.startsWith("./files/")||name.includes(".."))throw new Error("unexpected font path");
 const bytes=await readFile(path.join(fontRoot,name));css=css.replace(match[0],`url(data:font/woff2;base64,${bytes.toString("base64")})`);
}
const browser=await chromium.launch({headless:true});
try{
 const page=await browser.newPage({viewport:{width:850,height:1100}});
 await page.setContent(`<style>${css}*{box-sizing:border-box}body{margin:0;font-family:'Noto Sans KR Variable',sans-serif}.page{width:816px;height:1056px;padding:60px;break-after:page}img{width:696px;height:348px}p{font-size:24px}</style><div class="page"><h1>madi attachment text source</h1><p>Offline knowledge source alpha 2026</p><p>첫 페이지 원본 텍스트</p></div><div class="page"><img id="scan"></div><span style="position:fixed;visibility:hidden;font-size:64px">오프라인 문서 추출 검증</span>`);
 await page.evaluate(()=>document.fonts.ready);
 const data=await page.evaluate(()=>{
  const canvas=document.createElement("canvas");canvas.width=1392;canvas.height=696;const c=canvas.getContext("2d");c.fillStyle="white";c.fillRect(0,0,canvas.width,canvas.height);c.fillStyle="black";c.font="64px 'Noto Sans KR Variable',sans-serif";c.fillText("MADI OFFLINE OCR 2026",70,190);c.fillText("오프라인 문서 추출 검증",70,350);document.querySelector("#scan").src=canvas.toDataURL("image/png");return canvas.toDataURL("image/png");
 });
 await page.evaluate(()=>{document.querySelector("span").remove();document.querySelector(".page:last-child").style.breakAfter="auto"});
 await page.pdf({path:path.join(output,"source"),width:"816px",height:"1056px",printBackground:true,margin:{top:0,left:0,right:0,bottom:0},preferCSSPageSize:true});
 await writeFile(path.join(output,"expected-scan.png"),Buffer.from(data.split(",")[1],"base64"));
 console.log(JSON.stringify({output,pages:2,ocrPage:2,expectedEnglish:"MADI OFFLINE OCR 2026",expectedKorean:"오프라인 문서 추출 검증"}));
}finally{await browser.close()}
