import * as io from "@actions/io";
import * as path from "path";
import * as os from "os";
import * as fs from "fs";
import * as tc from "@actions/tool-cache";
import * as httpm from "@actions/http-client";
import nock from "nock";

const toolDir = path.join(__dirname, "runner", "tools");
const tempDir = path.join(__dirname, "runner", "temp");
const dataDir = path.join(__dirname, "testdata");
const IS_WINDOWS = process.platform === "win32";
const GITHUB_TOKEN = process.env.GITHUB_TOKEN || "";

process.env.RUNNER_TEMP = tempDir;
process.env.RUNNER_TOOL_CACHE = toolDir;
import * as installer from "../src/installer";

describe("filename tests", () => {
  const tests = [
    ["protoc-23.2-linux-x86_32.zip", "linux", ""],
    ["protoc-23.2-linux-x86_64.zip", "linux", "x64"],
    ["protoc-23.2-linux-aarch_64.zip", "linux", "arm64"],
    ["protoc-23.2-linux-ppcle_64.zip", "linux", "ppc64"],
    ["protoc-23.2-linux-s390_64.zip", "linux", "s390x"],
    ["protoc-23.2-osx-aarch_64.zip", "darwin", "arm64"],
    ["protoc-23.2-osx-x86_64.zip", "darwin", "x64"],
    ["protoc-23.2-win64.zip", "win32", "x64"],
    ["protoc-23.2-win32.zip", "win32", "x32"],
  ];
  it(`Downloads all expected versions correctly`, () => {
    for (const [expected, plat, arch] of tests) {
      const actual = installer.getFileName("23.2", plat, arch);
      expect(expected).toBe(actual);
    }
  });
});

describe("download retry tests", () => {
  afterEach(() => {
    jest.useRealTimers();
    jest.restoreAllMocks();
  });

  it("retries failed downloads twice before succeeding", async () => {
    jest.useFakeTimers();
    const downloadToolSpy = jest
      .spyOn(tc, "downloadTool")
      .mockRejectedValueOnce(new Error("first failure"))
      .mockRejectedValueOnce(new Error("second failure"))
      .mockResolvedValueOnce("/tmp/protoc.zip");

    const download = installer.downloadToolWithRetries(
      "https://example.com/protoc.zip",
    );

    await jest.advanceTimersByTimeAsync(10000);
    await jest.advanceTimersByTimeAsync(10000);

    await expect(download).resolves.toBe("/tmp/protoc.zip");
    expect(downloadToolSpy).toHaveBeenCalledTimes(3);
  });

  it("fails after the initial download attempt and two retries", async () => {
    jest.useFakeTimers();
    const downloadToolSpy = jest
      .spyOn(tc, "downloadTool")
      .mockRejectedValue(new Error("download failed"));

    const download = installer.downloadToolWithRetries(
      "https://example.com/protoc.zip",
    );
    const assertion = expect(download).rejects.toThrow("download failed");

    await jest.advanceTimersByTimeAsync(10000);
    await jest.advanceTimersByTimeAsync(10000);

    await assertion;
    expect(downloadToolSpy).toHaveBeenCalledTimes(3);
  });
});

describe("GitHub releases retry tests", () => {
  afterEach(() => {
    jest.useRealTimers();
    jest.restoreAllMocks();
  });

  it("retries failed GitHub releases requests twice before succeeding", async () => {
    jest.useFakeTimers();
    const releasePage = {
      statusCode: 200,
      result: [{ tag_name: "v23.1", prerelease: false }],
      headers: {},
    };
    const emptyPage = {
      statusCode: 200,
      result: [],
      headers: {},
    };
    const getSpy = jest
      .spyOn(httpm.HttpClient.prototype, "getJson")
      .mockRejectedValueOnce(
        new Error(
          "Request timeout: /repos/protocolbuffers/protobuf/releases?page=1",
        ),
      )
      .mockRejectedValueOnce(
        new Error(
          "Request timeout: /repos/protocolbuffers/protobuf/releases?page=1",
        ),
      )
      .mockResolvedValueOnce(releasePage)
      .mockResolvedValueOnce(emptyPage);
    const findSpy = jest.spyOn(tc, "find").mockReturnValue("/tmp/protoc");

    const protoc = installer.getProtoc("v23.1", false, "");

    await jest.advanceTimersByTimeAsync(10000);
    await jest.advanceTimersByTimeAsync(10000);

    await expect(protoc).resolves.toBeUndefined();
    expect(getSpy).toHaveBeenCalledTimes(4);
    expect(findSpy).toHaveBeenCalledWith("protoc", "v23.1");
  });

  it("fails GitHub releases requests after the initial attempt and two retries", async () => {
    jest.useFakeTimers();
    const getSpy = jest
      .spyOn(httpm.HttpClient.prototype, "getJson")
      .mockRejectedValue(new Error("GitHub API failed"));

    const protoc = installer.getProtoc("v23.1", false, "");
    const assertion = expect(protoc).rejects.toThrow("GitHub API failed");

    await jest.advanceTimersByTimeAsync(10000);
    await jest.advanceTimersByTimeAsync(10000);

    await assertion;
    expect(getSpy).toHaveBeenCalledTimes(3);
  });
});

describe("installer tests", () => {
  beforeEach(async function () {
    await io.rmRF(toolDir);
    await io.rmRF(tempDir);
    await io.mkdirP(toolDir);
    await io.mkdirP(tempDir);
  });

  afterAll(async () => {
    try {
      await io.rmRF(toolDir);
      await io.rmRF(tempDir);
    } catch {
      console.log("Failed to remove test directories");
    }
  });

  it("Downloads version of protoc if no matching version is installed", async () => {
    await installer.getProtoc("v23.0", true, GITHUB_TOKEN);
    const protocDir = path.join(toolDir, "protoc", "v23.0", os.arch());

    expect(fs.existsSync(`${protocDir}.complete`)).toBe(true);

    if (IS_WINDOWS) {
      expect(fs.existsSync(path.join(protocDir, "bin", "protoc.exe"))).toBe(
        true,
      );
    } else {
      expect(fs.existsSync(path.join(protocDir, "bin", "protoc"))).toBe(true);
    }
  }, 100000);

  describe("Gets the latest release of protoc", () => {
    beforeEach(() => {
      nock("https://api.github.com")
        .get("/repos/protocolbuffers/protobuf/releases?page=1")
        .replyWithFile(200, path.join(dataDir, "releases-1.json"));

      nock("https://api.github.com")
        .get("/repos/protocolbuffers/protobuf/releases?page=2")
        .replyWithFile(200, path.join(dataDir, "releases-2.json"));

      nock("https://api.github.com")
        .get("/repos/protocolbuffers/protobuf/releases?page=3")
        .replyWithFile(200, path.join(dataDir, "releases-3.json"));

      nock("https://api.github.com")
        .get("/repos/protocolbuffers/protobuf/releases?page=4")
        .replyWithFile(200, path.join(dataDir, "releases-4.json"));

      nock("https://api.github.com")
        .get("/repos/protocolbuffers/protobuf/releases?page=5")
        .replyWithFile(200, path.join(dataDir, "releases-5.json"));

      nock("https://api.github.com")
        .get("/repos/protocolbuffers/protobuf/releases?page=6")
        .replyWithFile(200, path.join(dataDir, "releases-6.json"));
    });

    afterEach(() => {
      nock.cleanAll();
      nock.enableNetConnect();
    });

    const tests = [
      ["v23.1", "v23.1"],
      ["v22.x", "v22.5"],
      ["v23.0-rc2", "v23.0-rc2"],
    ];
    tests.forEach(function (testCase) {
      const [input, expected] = testCase;
      it(`Gets latest version of protoc using ${input} and no matching version is installed`, async () => {
        await installer.getProtoc(input, true, GITHUB_TOKEN);
        const protocDir = path.join(toolDir, "protoc", expected, os.arch());

        expect(fs.existsSync(`${protocDir}.complete`)).toBe(true);
        if (IS_WINDOWS) {
          expect(fs.existsSync(path.join(protocDir, "bin", "protoc.exe"))).toBe(
            true,
          );
        } else {
          expect(fs.existsSync(path.join(protocDir, "bin", "protoc"))).toBe(
            true,
          );
        }
      }, 100000);
    });
  });
});
