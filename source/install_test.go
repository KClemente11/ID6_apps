package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

var hostFileOps string

func TestMain(m *testing.M) {
	os.Exit(func() int {
		dir, err := os.MkdirTemp("", "gauge-fileops-test-")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer os.RemoveAll(dir)
		hostFileOps = filepath.Join(dir, fileOpsName)
		cmd := exec.Command("cc", "-std=c99", "-Wall", "-Wextra", "-Werror", "-o", hostFileOps, "native/fileops.c")
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "build file operations test helper: %v\n%s", err, out)
			return 1
		}
		if err := loadTestPayload(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return m.Run()
	}())
}

func loadTestPayload() error {
	payload := fstest.MapFS{
		"payload/tools/" + fileOpsName: {Data: []byte("host-test helper")},
	}
	for _, app := range apps {
		prefix := "payload/" + app.name + "/"
		for _, name := range []string{app.name + ".manifest", app.name + ".xml"} {
			data, err := os.ReadFile(filepath.Join("native", "plugin", name))
			if err != nil {
				return err
			}
			payload[prefix+name] = &fstest.MapFile{Data: data}
		}
		for _, name := range []string{app.name + "_ALL_Images.zip", "lib" + app.name + ".so", app.worker} {
			payload[prefix+name] = &fstest.MapFile{Data: []byte(name)}
		}
	}
	bundled = payload
	return nil
}

type identifyOnly struct {
	version string
	t       *testing.T
}

func (r identifyOnly) run(cmd string, _ []byte, _ int) ([]byte, error) {
	r.t.Helper()
	if cmd != "cat /opt/sys/etc/nbt_version.txt" {
		r.t.Fatalf("unexpected command on blocked firmware: %s", cmd)
	}
	return []byte(r.version + "\r\n"), nil
}

func TestUnsupportedFirmwareStopsBeforePayloadCheck(t *testing.T) {
	for _, version := range []string{"NBTevo_H16205L", "NBTevo_M17345I", "NBTevo_N18032I", "NBTevo_Z24000I"} {
		got := inspect(identifyOnly{version, t}, vehicleInput{}, checkGateway)
		if got.Ready || got.Version != version || !strings.Contains(got.Result, "outside") {
			t.Errorf("%s: %+v", version, got)
		}
	}
	got := inspect(identifyOnly{"NBTevo_N18092I", t}, vehicleInput{Version: "NBTevo_Y24073J"}, checkGateway)
	if got.Ready || !strings.Contains(got.Result, "differs") {
		t.Fatal(got)
	}
}

type shellFixture struct {
	dir               string
	failMapRename     bool
	corruptStagedRead bool
}

func (f shellFixture) run(command string, input []byte, _ int) ([]byte, error) {
	// Simulate the head unit with real local shell/filesystem semantics.
	const missingUtilities = "printf() { return 126; }\nls() { return 126; }\nrmdir() { echo 'rmdir: cannot execute - No such file or directory' >&2; return 126; }\nmv() { return 126; }\n"
	cmd := strings.ReplaceAll(command, root, f.dir)
	if strings.HasSuffix(command, "/"+fileOpsName+" check") {
		start := strings.LastIndex(cmd, " && ")
		cmd = cmd[:start] + " && " + hostFileOps + " check"
	}
	if f.corruptStagedRead && strings.HasPrefix(command, "cat ") && strings.HasSuffix(command, "/new-BoostDeck/BoostDeck.xml") {
		return []byte("truncated upload"), nil
	}
	if strings.HasPrefix(command, "find ") {
		cmd += " | sed 's|" + f.dir + "|" + root + "|'"
	}
	if command == "/bin/sh -s" {
		input = bytes.ReplaceAll(input, []byte(root), []byte(f.dir))
		input = bytes.ReplaceAll(input, []byte(`"$ops"`), []byte("test_fileops"))
		guard := ""
		if f.failMapRename {
			guard = `case "$1:$2:$3" in move:*/new-MhdMaps:*/Plugins/MhdMaps) return 1;; esac; `
		}
		input = append([]byte(missingUtilities+"test_fileops() { "+guard+hostFileOps+" \"$@\"; }\n"), input...)
	}
	process := exec.Command("sh", "-c", missingUtilities+cmd)
	process.Stdin = bytes.NewReader(input)
	out, err := process.Output()
	if failure, ok := err.(*exec.ExitError); ok {
		return out, fmt.Errorf("%w: %s", err, bytes.TrimSpace(failure.Stderr))
	}
	return out, err
}

func TestInstallStagesThreeAndRestoresAfterFailedRename(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(fmt.Sprint(failure), func(t *testing.T) {
			dir := t.TempDir()
			old := filepath.Join(dir, "Plugins", "BoostDeck")
			if err := os.MkdirAll(old, 0755); err != nil {
				t.Fatal(err)
			}
			manifest := []byte("<plugin><name>BoostDeck</name></plugin>")
			if err := os.WriteFile(filepath.Join(old, "BoostDeck.manifest"), manifest, 0644); err != nil {
				t.Fatal(err)
			}
			tx, err := install(shellFixture{dir: dir, failMapRename: failure}, "169.254.21.3", panel{1440, 540}, false)
			if failure {
				if err == nil || !strings.Contains(err.Error(), "Could not publish MhdMaps") {
					t.Fatalf("expected rollback, got %v", err)
				}
				got, e := os.ReadFile(filepath.Join(old, "BoostDeck.manifest"))
				if e != nil || !bytes.Equal(got, manifest) {
					t.Fatal("original plugin was not restored", e)
				}
				for _, name := range []string{"ShiftDeck", "MhdMaps"} {
					if _, e := os.Stat(filepath.Join(dir, "Plugins", name)); !os.IsNotExist(e) {
						t.Fatalf("%s was published", name)
					}
				}
				if _, e := os.Stat(filepath.Join(dir, "GaugeDeck-install-lock")); !os.IsNotExist(e) {
					t.Fatalf("rollback left an install lock: %v", e)
				}
				if _, e := install(shellFixture{dir: dir}, "169.254.21.3", panel{1440, 540}, false); e != nil {
					t.Fatalf("retry after rollback: %v", e)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			backup := filepath.Join(dir, filepath.Base(tx), "old-BoostDeck", "BoostDeck.manifest")
			got, e := os.ReadFile(backup)
			if e != nil || !bytes.Equal(got, manifest) {
				t.Fatal("backup was not preserved", e)
			}
			if _, e := os.Stat(filepath.Join(dir, "Plugins", "EngineCodes")); !os.IsNotExist(e) {
				t.Fatalf("unrelated EngineCodes menu was installed: %v", e)
			}
			for _, app := range apps {
				live := filepath.Join(dir, "Plugins", app.name)
				got, e := os.ReadFile(filepath.Join(live, "gateway-ip.txt"))
				if e != nil || string(got) != "169.254.21.3\n" {
					t.Fatalf("wrong gateway for %s", app.name)
				}
				xml, e := os.ReadFile(filepath.Join(live, app.name+".xml"))
				base, _ := fs.ReadFile(bundled, "payload/"+app.name+"/"+app.name+".xml")
				if e != nil || bytes.Equal(xml, base) || !bytes.Equal(xml, scaleLayout(base, app.name, panel{1440, 540})) {
					t.Fatal("expected the scaled wide layout", e)
				}
				f, e := os.Stat(filepath.Join(live, app.worker))
				if e != nil || f.Mode().Perm()&0111 == 0 {
					t.Fatalf("%s is not executable", app.worker)
				}
			}
		})
	}
}

func TestStagedReadbackMismatchLeavesLivePluginsAlone(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "Plugins", "BoostDeck")
	if err := os.MkdirAll(live, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := []byte("<plugin><name>BoostDeck</name></plugin>")
	path := filepath.Join(live, "BoostDeck.manifest")
	if err := os.WriteFile(path, manifest, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := install(shellFixture{dir: dir, corruptStagedRead: true}, "169.254.21.3", panel{1280, 480}, false)
	if err == nil || !strings.Contains(err.Error(), "upload verification failed for BoostDeck.xml") {
		t.Fatalf("expected upload verification failure, got %v", err)
	}
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(actual, manifest) {
		t.Fatal("existing plugin was changed", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "GaugeDeck-install-lock")); !os.IsNotExist(err) {
		t.Fatal("publication started after a failed staged readback", err)
	}
}

func TestPublicationLeavesAnExistingLockAlone(t *testing.T) {
	dir := t.TempDir()
	lock := filepath.Join(dir, "GaugeDeck-install-lock")
	if err := os.Mkdir(lock, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := shellFixture{dir: dir}
	if _, err := fixture.run("/bin/sh -s", publicationScript(root+"/GaugeDeck-test"), 1024); err == nil || !strings.Contains(err.Error(), "lock already exists") {
		t.Fatalf("expected a lock error, got %v", err)
	}
	if _, err := os.Stat(lock); err != nil {
		t.Fatal(err)
	}
}

func TestBlockedInstallReportsPreviousTransactionWithoutChanges(t *testing.T) {
	dir := t.TempDir()
	lock := filepath.Join(dir, "GaugeDeck-install-lock")
	tx := filepath.Join(dir, "GaugeDeck-ae68da9e4a9d7ec9")
	for _, path := range []string{lock, filepath.Join(tx, "old-BoostDeck"), filepath.Join(tx, "new-ShiftDeck"), filepath.Join(dir, "Plugins", "BoostDeck")} {
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
	}
	marker := filepath.Join(tx, "fresh-MhdMaps")
	if err := os.WriteFile(marker, nil, 0644); err != nil {
		t.Fatal(err)
	}
	newTx, err := install(shellFixture{dir: dir}, "169.254.21.3", panel{1280, 480}, true)
	if newTx != "" || err == nil {
		t.Fatalf("expected a read-only lock report, got %q, %v", newTx, err)
	}
	for _, line := range []string{"Lock type: directory", "Live app paths:", "BoostDeck: directory", "GaugeDeck-ae68da9e4a9d7ec9", "old-BoostDeck: directory", "new-ShiftDeck: directory", "fresh-MhdMaps: file"} {
		if !strings.Contains(err.Error(), line) {
			t.Fatalf("report missing %q: %v", line, err)
		}
	}
	for _, path := range []string{lock, filepath.Join(tx, "old-BoostDeck"), marker} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
	transactions, err := filepath.Glob(filepath.Join(dir, "GaugeDeck-????????????????"))
	if err != nil || len(transactions) != 1 || transactions[0] != tx {
		t.Fatalf("blocked attempt created another transaction: %v, %v", transactions, err)
	}
}

func TestStagedOnlyLockRecoveryInstallsAllApps(t *testing.T) {
	dir := t.TempDir()
	lock := filepath.Join(dir, "GaugeDeck-install-lock")
	previous := filepath.Join(dir, "GaugeDeck-0123456789abcdef")
	for _, path := range []string{lock, filepath.Join(dir, "Plugins")} {
		if err := os.Mkdir(path, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, app := range apps {
		if err := os.MkdirAll(filepath.Join(previous, "new-"+app.name), 0755); err != nil {
			t.Fatal(err)
		}
	}
	partial := filepath.Join(dir, "GaugeDeck-fedcba9876543210", "new-BoostDeck")
	if err := os.MkdirAll(partial, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := shellFixture{dir: dir}
	_, err := install(fixture, "169.254.21.3", panel{1280, 480}, false)
	blocked, ok := err.(*blockedInstall)
	if !ok || !blocked.canRecover {
		t.Fatalf("expected a recoverable staged transaction: %v", err)
	}
	// Recheck on confirmation. A file appearing in the lock rules out recovery.
	guard := filepath.Join(lock, ".owner")
	if err := os.WriteFile(guard, []byte("active"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = install(fixture, "169.254.21.3", panel{1280, 480}, true)
	if blocked, ok := err.(*blockedInstall); !ok || blocked.canRecover {
		t.Fatalf("expected a nonempty lock to block recovery: %v", err)
	}
	if err := os.Remove(guard); err != nil {
		t.Fatal(err)
	}
	if _, err := install(fixture, "169.254.21.3", panel{1280, 480}, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(lock); !os.IsNotExist(err) {
		t.Fatal("install lock remains", err)
	}
	for _, app := range apps {
		if _, err := os.Stat(filepath.Join(previous, "new-"+app.name)); err != nil {
			t.Fatal("previous staging was changed", err)
		}
		if worker, err := os.Stat(filepath.Join(dir, "Plugins", app.name, app.worker)); err != nil || worker.Mode().Perm()&0111 == 0 {
			t.Fatal("new worker is missing or not executable", err)
		}
	}
}

type compatibilityFixture struct {
	identifyOnly
	app, core []byte
	display   string
}

func (r compatibilityFixture) run(cmd string, _ []byte, _ int) ([]byte, error) {
	switch cmd {
	case "cat /opt/sys/etc/nbt_version.txt":
		return []byte(r.version), nil
	case "uname -a":
		return []byte("QNX hu-omap 6.5.0 2015/12/01 TI-OMAP5432-uEVM armle\n"), nil
	case "cat /opt/hmi/ID5/lib/" + appFramework:
		return r.app, nil
	case "cat /opt/hmi/ID5/lib/" + coreFramework:
		return r.core, nil
	case "/opt/sys/bin/LayerManagerControl get screen 0":
		return []byte(r.display), nil
	}
	if strings.HasPrefix(cmd, "for f in ") {
		return []byte("#Coded is: def,1180_480,touch_screen\n"), nil
	}
	if strings.HasPrefix(cmd, "test -d "+root+"/Plugins") {
		return nil, nil
	}
	r.t.Fatalf("unexpected check: %s", cmd)
	return nil, nil
}

func TestInspectionRequiresMatchingFrameworksAndActivePanel(t *testing.T) {
	const version = "NBTevo_V20153I"
	fixture := compatibilityFixture{
		identifyOnly: identifyOnly{version, t},
		app:          []byte("app framework"), core: []byte("core framework"),
		display: "- resolution: x=1440, y=540\n",
	}
	saved := profiles[version]
	profiles[version] = frameworkPair{digest(fixture.app), digest(fixture.core)}
	defer func() { profiles[version] = saved }()
	input := vehicleInput{Head: "169.254.21.10", Gateway: "169.254.21.3"}
	reportsHead := func(_, _ string) (string, error) { return input.Head, nil }
	got := inspect(fixture, input, reportsHead)
	if !got.Ready || got.screen != (panel{1440, 540}) || !strings.Contains(got.Touch, "Touch coded") {
		t.Fatal(got)
	}
	mismatch := fixture
	mismatch.core = []byte("other core")
	if got := inspect(mismatch, input, reportsHead); got.Ready || !strings.Contains(got.Result, coreFramework) {
		t.Fatal(got)
	}
	fixture.display = ""
	if got := inspect(fixture, input, reportsHead); got.Ready || got.Display != "Unknown" {
		t.Fatal(got)
	}
	input.DisplayResolution = "1280x480"
	if got := inspect(fixture, input, reportsHead); !got.Ready || got.screen != (panel{1280, 480}) || !strings.Contains(got.Display, "manual") {
		t.Fatal(got)
	}
	if got := inspect(mismatch, input, reportsHead); got.Ready {
		t.Fatal(got)
	}
	fixture.display = "- resolution: x=1920, y=720\n"
	if got := inspect(fixture, input, reportsHead); got.Ready || !strings.Contains(got.Result, "differs") {
		t.Fatal(got)
	}
	input.DisplayResolution = "1920x720"
	if got := inspect(fixture, input, reportsHead); !got.Ready || got.screen != (panel{1920, 720}) || !strings.Contains(got.Display, "Experimental") {
		t.Fatal(got)
	}
	input.DisplayResolution = ""
	if got := inspect(fixture, input, reportsHead); got.Ready || got.Display != "1920 × 720" {
		t.Fatal(got)
	}
	fixture.display = ""
	input.DisplayResolution = "1600x600"
	if got := inspect(fixture, input, reportsHead); !got.Ready || got.screen != (panel{1600, 600}) {
		t.Fatal(got)
	}
	input.DisplayResolution = "800x720"
	if got := inspect(fixture, input, reportsHead); got.Ready || !strings.Contains(got.Result, "aspect ratio") {
		t.Fatal(got)
	}
	input.DisplayResolution = ""
	fixture.display = "- resolution: x=800, y=480\n"
	if got := inspect(fixture, input, reportsHead); !got.Ready || got.screen != (panel{800, 480}) {
		t.Fatal(got)
	}
	if got := inspect(fixture, input, func(_, _ string) (string, error) { return "", fmt.Errorf("connection refused") }); got.Ready || !strings.Contains(got.Result, "port 6801") {
		t.Fatal(got)
	}
	if got := inspect(fixture, input, func(_, _ string) (string, error) { return "169.254.21.100", nil }); got.Ready || !strings.Contains(got.Result, "different head unit") {
		t.Fatal(got)
	}
}
