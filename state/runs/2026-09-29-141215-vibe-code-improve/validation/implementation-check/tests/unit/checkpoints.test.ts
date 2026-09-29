import { execFileSync } from "node:child_process";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { createCheckpoint, isDestructiveCommand } from "../../src/features/checkpoints";

const temps: string[] = [];

function tempDir(): string {
	const dir = fs.mkdtempSync(path.join(os.tmpdir(), "vibe-checkpoint-test-"));
	temps.push(dir);
	return fs.realpathSync(dir);
}

function git(cwd: string, args: string[]): string {
	return execFileSync("git", args, { cwd, encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] }).trim();
}

/** A real git repository — the checkpoint contract is about git behaviour, so nothing here is faked. */
function repo(withCommit = true): string {
	const dir = tempDir();
	git(dir, ["init", "-q", "."]);
	git(dir, ["config", "user.email", "test@example.com"]);
	git(dir, ["config", "user.name", "checkpoint test"]);
	if (withCommit) {
		fs.writeFileSync(path.join(dir, "tracked.txt"), "one\n", "utf8");
		git(dir, ["add", "-A"]);
		git(dir, ["commit", "-qm", "init"]);
	}
	return dir;
}

afterEach(() => {
	while (temps.length > 0) fs.rmSync(temps.pop() as string, { recursive: true, force: true });
});

describe("createCheckpoint", () => {
	it("snapshots uncommitted tracked changes and untracked files, skipping ignored ones", () => {
		const ws = repo();
		fs.writeFileSync(path.join(ws, "tracked.txt"), "one\ntwo\n", "utf8");
		fs.writeFileSync(path.join(ws, "untracked.txt"), "new file\n", "utf8");
		fs.mkdirSync(path.join(ws, "nested"), { recursive: true });
		fs.writeFileSync(path.join(ws, "nested", "deep.txt"), "nested new file\n", "utf8");
		fs.writeFileSync(path.join(ws, ".gitignore"), "ignored.txt\n", "utf8");
		fs.writeFileSync(path.join(ws, "ignored.txt"), "junk\n", "utf8");

		const cp = createCheckpoint(ws, path.join(ws, ".vibe-code", "checkpoints"), "rm -rf build", "테스트");

		expect(cp.commit).toBeTruthy();
		const files = git(ws, ["ls-tree", "-r", "--name-only", cp.commit as string]).split("\n");
		expect(files).toContain("tracked.txt");
		expect(files).toContain("untracked.txt");
		expect(files).toContain("nested/deep.txt");
		expect(files).not.toContain("ignored.txt");
		expect(git(ws, ["show", `${cp.commit}:tracked.txt`])).toBe("one\ntwo");
		expect(git(ws, ["show", `${cp.commit}:untracked.txt`])).toBe("new file");
	});

	it("leaves the working tree, index and stash stack untouched", () => {
		const ws = repo();
		fs.writeFileSync(path.join(ws, "tracked.txt"), "one\ntwo\n", "utf8");
		fs.writeFileSync(path.join(ws, "staged.txt"), "staged\n", "utf8");
		git(ws, ["add", "staged.txt"]);
		fs.writeFileSync(path.join(ws, "untracked.txt"), "new file\n", "utf8");
		const before = git(ws, ["status", "--porcelain"]);

		// The note is written outside the work tree so that only the snapshot itself is under test.
		const cp = createCheckpoint(ws, path.join(tempDir(), "checkpoints"), "git reset --hard", "테스트");

		expect(git(ws, ["status", "--porcelain"])).toBe(before);
		expect(git(ws, ["stash", "list"])).toBe("");
		expect(fs.readFileSync(path.join(ws, "tracked.txt"), "utf8")).toBe("one\ntwo\n");
		expect(git(ws, ["rev-parse", cp.ref as string])).toBe(cp.commit);
	});

	it("does not leave its temporary index behind in the checkpoints directory", () => {
		const ws = repo();
		fs.writeFileSync(path.join(ws, "untracked.txt"), "new file\n", "utf8");
		const dir = path.join(ws, ".vibe-code", "checkpoints");

		const cp = createCheckpoint(ws, dir, "rm -rf build", "테스트");

		expect(fs.readdirSync(dir)).toEqual([path.basename(cp.file)]);
	});

	it("writes a restore block whose commands actually run against the snapshot", () => {
		const ws = repo();
		fs.writeFileSync(path.join(ws, "tracked.txt"), "one\ntwo\n", "utf8");
		fs.writeFileSync(path.join(ws, "untracked.txt"), "new file\n", "utf8");

		const cp = createCheckpoint(ws, path.join(ws, ".vibe-code", "checkpoints"), "rm -rf build", "테스트");
		const note = fs.readFileSync(cp.file, "utf8");

		// `git stash apply` only accepts stash-like commits; a plain snapshot commit is not one.
		expect(note).not.toContain("git stash apply");
		expect(note).toContain("스냅샷에 없던 파일");
		const commands = note
			.split("\n")
			.filter((line) => line.startsWith("git "))
			.map((line) => line.replace(/\s+#.*$/, "").trim());
		expect(commands.length).toBeGreaterThanOrEqual(2);

		// Destroy the files the way the destructive command would have, then run the documented restore.
		fs.rmSync(path.join(ws, "tracked.txt"));
		fs.rmSync(path.join(ws, "untracked.txt"));
		for (const command of commands) git(ws, command.split(/\s+/).slice(1));
		expect(fs.readFileSync(path.join(ws, "tracked.txt"), "utf8")).toBe("one\ntwo\n");
		expect(fs.readFileSync(path.join(ws, "untracked.txt"), "utf8")).toBe("new file\n");
	});

	it("reports a clean working tree without pinning a ref", () => {
		const ws = repo();

		const cp = createCheckpoint(ws, path.join(ws, ".vibe-code", "checkpoints"), "(manual)", "테스트");

		expect(cp.commit).toBeFalsy();
		expect(cp.ref).toBeUndefined();
		expect(cp.note).toContain("깨끗해서");
		expect(git(ws, ["for-each-ref", "refs/vibe-checkpoints"])).toBe("");
	});

	it("snapshots a repository that has no commits yet", () => {
		const ws = repo(false);
		fs.writeFileSync(path.join(ws, "untracked.txt"), "new file\n", "utf8");

		const cp = createCheckpoint(ws, path.join(tempDir(), "checkpoints"), "rm -rf .", "테스트");

		expect(cp.commit).toBeTruthy();
		expect(git(ws, ["ls-tree", "-r", "--name-only", cp.commit as string]).split("\n")).toContain("untracked.txt");
		expect(git(ws, ["status", "--porcelain"])).toBe("?? untracked.txt");
	});

	it("still writes a note outside a git repository", () => {
		const ws = tempDir();
		const dir = path.join(ws, ".vibe-code", "checkpoints");

		const cp = createCheckpoint(ws, dir, "rm -rf build", "테스트");

		expect(cp.commit).toBeUndefined();
		expect(cp.ref).toBeUndefined();
		expect(cp.note).toContain("git 저장소가 아니어서");
		expect(fs.readFileSync(cp.file, "utf8")).toContain("명령: `rm -rf build`");
		expect(fs.readdirSync(dir)).toEqual([path.basename(cp.file)]);
	});
});

describe("isDestructiveCommand", () => {
	it("flags deleting, history-rewriting and production-hitting commands", () => {
		for (const command of ["rm -rf build", "git reset --hard", "git push origin main --force", "kubectl delete pod x", "DROP TABLE users"]) {
			expect(isDestructiveCommand(command), command).toBe(true);
		}
	});

	it("leaves ordinary commands alone", () => {
		for (const command of ["npm test", "git status", "ls -R", "git restore --staged file.ts"]) {
			expect(isDestructiveCommand(command), command).toBe(false);
		}
	});
});
