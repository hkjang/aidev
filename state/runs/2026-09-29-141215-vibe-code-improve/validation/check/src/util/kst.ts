const KST_FORMAT = new Intl.DateTimeFormat("ko-KR", {
	timeZone: "Asia/Seoul",
	year: "numeric",
	month: "2-digit",
	day: "2-digit",
	hour: "2-digit",
	minute: "2-digit",
	second: "2-digit",
	hour12: false,
});

export interface KstParts {
	year: string;
	month: string;
	day: string;
	hour: string;
	minute: string;
	second: string;
}

export function kstParts(date: Date = new Date()): KstParts {
	const parts: Record<string, string> = {};
	for (const part of KST_FORMAT.formatToParts(date)) parts[part.type] = part.value;
	return {
		year: parts.year ?? "",
		month: parts.month ?? "",
		day: parts.day ?? "",
		hour: parts.hour ?? "",
		minute: parts.minute ?? "",
		second: parts.second ?? "",
	};
}

/** `YYYY-MM-DD` in Asia/Seoul. */
export function kstDate(date: Date = new Date()): string {
	const p = kstParts(date);
	return `${p.year}-${p.month}-${p.day}`;
}

/** `HH:MM KST` in Asia/Seoul. */
export function kstClock(date: Date = new Date()): string {
	const p = kstParts(date);
	return `${p.hour}:${p.minute} KST`;
}

export interface Stamp {
	/** Filename-safe: `YYYY-MM-DD-HHMMSS` */
	file: string;
	/** Human readable: `YYYY-MM-DD HH:MM:SS KST` */
	human: string;
}

export function kstStamp(date: Date = new Date()): Stamp {
	const p = kstParts(date);
	return {
		file: `${p.year}-${p.month}-${p.day}-${p.hour}${p.minute}${p.second}`,
		human: `${p.year}-${p.month}-${p.day} ${p.hour}:${p.minute}:${p.second} KST`,
	};
}
