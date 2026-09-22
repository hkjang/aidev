import { lstat, readdir, readFile, writeFile } from "node:fs/promises";
import { resolve, relative, sep } from "node:path";
import { pathToFileURL } from "node:url";

const crcTable = Array.from({ length: 256 }, (_, n) => {
  let crc = n;
  for (let bit = 0; bit < 8; bit++)
    crc = crc & 1 ? 0xedb88320 ^ (crc >>> 1) : crc >>> 1;
  return crc >>> 0;
});
function crc32(data) {
  let crc = 0xffffffff;
  for (const byte of data) crc = crcTable[(crc ^ byte) & 255] ^ (crc >>> 8);
  return (crc ^ 0xffffffff) >>> 0;
}
/** Deterministic ZIP_STORED packager using Node only; no registry or zip CLI. */
export async function createPluginArchive(directory) {
  const root = resolve(directory),
    files = [];
  async function visit(dir) {
    for (const entry of (await readdir(dir)).sort()) {
      const target = resolve(dir, entry),
        info = await lstat(target);
      if (info.isSymbolicLink())
        throw new Error("심볼릭 링크는 패키지에 포함할 수 없습니다");
      if (info.isDirectory()) await visit(target);
      else if (info.isFile()) {
        if (info.size > 4 * 1024 * 1024)
          throw new Error("파일당 최대 4MB입니다");
        files.push({
          name: relative(root, target).split(sep).join("/"),
          data: await readFile(target),
        });
      } else throw new Error("일반 파일만 패키지에 포함할 수 있습니다");
    }
  }
  await visit(root);
  if (files.length > 200) throw new Error("최대 200개 파일을 지원합니다");
  const chunks = [],
    central = [];
  let offset = 0;
  for (const file of files) {
    const name = Buffer.from(file.name),
      crc = crc32(file.data);
    const header = Buffer.alloc(30);
    header.writeUInt32LE(0x04034b50);
    header.writeUInt16LE(20, 4);
    header.writeUInt16LE(0x800, 6);
    header.writeUInt32LE(crc, 14);
    header.writeUInt32LE(file.data.length, 18);
    header.writeUInt32LE(file.data.length, 22);
    header.writeUInt16LE(name.length, 26);
    chunks.push(header, name, file.data);
    const record = Buffer.alloc(46);
    record.writeUInt32LE(0x02014b50);
    record.writeUInt16LE(20, 4);
    record.writeUInt16LE(20, 6);
    record.writeUInt16LE(0x800, 8);
    record.writeUInt32LE(crc, 16);
    record.writeUInt32LE(file.data.length, 20);
    record.writeUInt32LE(file.data.length, 24);
    record.writeUInt16LE(name.length, 28);
    record.writeUInt32LE(offset, 42);
    central.push(record, name);
    offset += header.length + name.length + file.data.length;
  }
  const centralData = Buffer.concat(central),
    end = Buffer.alloc(22);
  end.writeUInt32LE(0x06054b50);
  end.writeUInt16LE(files.length, 8);
  end.writeUInt16LE(files.length, 10);
  end.writeUInt32LE(centralData.length, 12);
  end.writeUInt32LE(offset, 16);
  const result = Buffer.concat([...chunks, centralData, end]);
  if (result.length > 10 * 1024 * 1024)
    throw new Error("ZIP은 최대 10MB입니다");
  return result;
}
if (
  process.argv[1] &&
  import.meta.url === pathToFileURL(resolve(process.argv[1])).href
) {
  const [, , directory, output] = process.argv;
  if (!directory || !output) {
    console.error(
      "사용법: node sdk/plugins/package.mjs <플러그인 폴더> <출력.zip>",
    );
    process.exitCode = 1;
  } else {
    try {
      const archive = await createPluginArchive(directory);
      await writeFile(resolve(output), archive, { flag: "wx" });
      console.log(`패키지 생성: ${output} (${archive.length} bytes)`);
    } catch (error) {
      console.error(error.message);
      process.exitCode = 1;
    }
  }
}
