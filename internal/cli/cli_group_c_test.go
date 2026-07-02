package cli

import (
	"path/filepath"
	"testing"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/internal/sources"
	"github.com/reeinharrrd/maestro/pkg/models"
)

func TestQueryCmd_EmptyDB(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newQueryCmd(&dbPath)
	if err := cmd.RunE(cmd, []string{"SELECT count(*) FROM providers"}); err != nil {
		t.Fatal(err)
	}
}

func TestQueryCmd_Seeded(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertProvider(&models.Provider{ID: "tp", Name: "Test Provider", Status: "active", Source: "test"}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newQueryCmd(&dbPath)
	if err := cmd.RunE(cmd, []string{"SELECT id, name FROM providers ORDER BY id"}); err != nil {
		t.Fatal(err)
	}
}

func TestQueryCmd_NoArgs(t *testing.T) {
	t.Parallel()
	cmd := newQueryCmd(nil)
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected error for missing query")
	}
}

func TestQueryCmd_MultiRow(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"p1", "p2", "p3"} {
		if err := d.UpsertProvider(&models.Provider{ID: id, Name: "P-" + id, Status: "active", Source: "test"}); err != nil {
			t.Fatal(err)
		}
	}
	d.Close()

	cmd := newQueryCmd(&dbPath)
	if err := cmd.RunE(cmd, []string{"SELECT id FROM providers ORDER BY id"}); err != nil {
		t.Fatal(err)
	}
}

func TestRouteCmd_ListEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newRouteCmd(&dbPath)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestRouteCmd_ListWithRules(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertRoutingRule(&models.RoutingRule{
		TaskKey: "coding_fast", CurrentModelID: "gpt-4o-mini",
		Description: "Fast coding", NeedsFC: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertRoutingRule(&models.RoutingRule{
		TaskKey: "reasoning", CurrentModelID: "o1",
		Description: "Deep reasoning",
	}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newRouteCmd(&dbPath)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestRouteCmd_TaskFlag(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertProvider(&models.Provider{ID: "tp", Name: "Test Provider", Status: "active", Source: "test"}); err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertModel(&models.Model{
		ID: "test-model", ProviderID: "tp", DisplayName: "Test Model",
		Status: "active", Source: "test", ContextWindow: 200000,
		FunctionCalling: true, LatencyP50Ms: 100,
		PricingPrompt: 0, PricingCompletion: 0, Tier: "free",
	}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newRouteCmd(&dbPath)
	cmd.Flags().Set("task", "coding_complex")
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestRouteCmd_TaskFallbackChain(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertProvider(&models.Provider{ID: "tp", Name: "Test Provider", Status: "active", Source: "test"}); err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertModel(&models.Model{
		ID: "fast-model", ProviderID: "tp", DisplayName: "Fast",
		Status: "active", Source: "test", ContextWindow: 1000,
		FunctionCalling: false, LatencyP50Ms: 30,
		PricingPrompt: 0, PricingCompletion: 0, Tier: "free",
	}); err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertModel(&models.Model{
		ID: "big-model", ProviderID: "tp", DisplayName: "Big",
		Status: "active", Source: "test", ContextWindow: 200000,
		FunctionCalling: true, LatencyP50Ms: 800,
		PricingPrompt: 0.01, PricingCompletion: 0.01, Tier: "free",
	}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newRouteCmd(&dbPath)
	cmd.Flags().Set("task", "fastest")
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestRouteCmd_ReportEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newRouteReportCmd(&dbPath)
	cmd.Flags().Set("limit", "5")
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestRouteCmd_ReportWithEvents(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.InsertRoutingEvent(&models.RoutingEvent{
		TaskKey: "coding_fast", SelectedModel: "m1",
		Reason: "selected by test", Shadow: false,
	}); err != nil {
		t.Fatal(err)
	}
	if err := d.InsertRoutingEvent(&models.RoutingEvent{
		TaskKey: "reasoning", SelectedModel: "m2",
		Reason: "shadow test", Shadow: true,
	}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newRouteReportCmd(&dbPath)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSourcesCmd_Subcommands(t *testing.T) {
	t.Parallel()
	cmd := newSourcesCmd(nil)
	for _, name := range []string{"list", "add", "remove", "sync", "discover", "install", "uninstall"} {
		sub, _, _ := cmd.Find([]string{name})
		if sub == nil {
			t.Errorf("expected subcommand %q", name)
		}
	}
}

func TestSourcesCmd_ListEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSourcesCmd(&dbPath)
	listCmd, _, _ := cmd.Find([]string{"list"})
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSourcesCmd_ListWithSources(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertSource(&models.Source{
		ID: "gh-user-repo", RemoteURL: "https://github.com/user/repo.git",
		Status: "active", LastSynced: 1700000000,
	}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSourcesCmd(&dbPath)
	listCmd, _, _ := cmd.Find([]string{"list"})
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSourcesCmd_RemoveNonexistent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSourcesCmd(&dbPath)
	removeCmd, _, _ := cmd.Find([]string{"remove"})
	if err := removeCmd.RunE(removeCmd, []string{"nonexistent"}); err == nil {
		t.Fatal("expected error for nonexistent source")
	}
}

func TestSourcesCmd_RemoveSeeded(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertSource(&models.Source{
		ID: "to-remove", RemoteURL: "https://example.com/repo.git",
		Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSourcesCmd(&dbPath)
	removeCmd, _, _ := cmd.Find([]string{"remove"})
	if err := removeCmd.RunE(removeCmd, []string{"to-remove"}); err != nil {
		t.Fatal(err)
	}
}

func TestSourcesCmd_SyncNonexistent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSourcesCmd(&dbPath)
	syncCmd, _, _ := cmd.Find([]string{"sync"})
	if err := syncCmd.RunE(syncCmd, []string{"nonexistent-src"}); err == nil {
		t.Fatal("expected error syncing nonexistent source")
	}
}

func TestSourcesCmd_DiscoverNonexistent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSourcesCmd(&dbPath)
	discoverCmd, _, _ := cmd.Find([]string{"discover"})
	if err := discoverCmd.RunE(discoverCmd, []string{"nonexistent"}); err == nil {
		t.Fatal("expected error discovering nonexistent source")
	}
}

func TestSourceItemsCmd_Subcommands(t *testing.T) {
	t.Parallel()
	cmd := newSourceItemsCmd(nil)
	for _, name := range []string{"list", "import", "report"} {
		sub, _, _ := cmd.Find([]string{name})
		if sub == nil {
			t.Errorf("expected subcommand %q", name)
		}
	}
}

func TestSourceItemsCmd_ListEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSourceItemsCmd(&dbPath)
	listCmd, _, _ := cmd.Find([]string{"list"})
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSourceItemsCmd_ListWithItems(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertSource(&models.Source{ID: "src1", RemoteURL: "https://github.com/test", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertSourceItem(&models.SourceItem{
		ID: "src-item-1", SourceID: "src1", Type: "skill", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertSourceItem(&models.SourceItem{
		ID: "src-item-2", SourceID: "src1", Type: "agent", Status: "installed",
	}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSourceItemsCmd(&dbPath)
	listCmd, _, _ := cmd.Find([]string{"list"})
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSourceItemsCmd_ReportEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSourceItemsCmd(&dbPath)
	reportCmd, _, _ := cmd.Find([]string{"report"})
	if err := reportCmd.RunE(reportCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSourceItemsCmd_ReportWithItems(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertSource(&models.Source{ID: "s1", RemoteURL: "https://github.com/test", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertSourceItem(&models.SourceItem{
		ID: "si-1", SourceID: "s1", Type: "skill", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSourceItemsCmd(&dbPath)
	cmd.SetArgs([]string{"report"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestDaemonCmd_Constructs(t *testing.T) {
	t.Parallel()
	cmd := newDaemonCmd(nil)
	if cmd == nil {
		t.Fatal("expected daemon command")
	}
	if cmd.Use != "daemon" {
		t.Errorf("expected Use='daemon', got %q", cmd.Use)
	}
	interval := cmd.Flags().Lookup("interval")
	if interval == nil {
		t.Fatal("expected --interval flag")
	}
	if interval.DefValue != "5m0s" {
		t.Errorf("default interval = %q, want 5m0s", interval.DefValue)
	}
	install := cmd.Flags().Lookup("install")
	if install == nil {
		t.Fatal("expected --install flag")
	}
	if install.DefValue != "false" {
		t.Errorf("default install = %q, want false", install.DefValue)
	}
}

func TestDaemonCmd_RunSyncCycle_NoSources(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	svc := sources.New(d)
	inst := sources.NewInstaller(d)

	runSyncCycle(svc, inst, false)
	d.Close()
}

func TestCompressCmd_Subcommands(t *testing.T) {
	t.Parallel()
	cmd := newCompressCmd()
	for _, name := range []string{"demo", "report", "save", "prune"} {
		sub, _, _ := cmd.Find([]string{name})
		if sub == nil {
			t.Errorf("expected subcommand %q", name)
		}
	}
}

func TestCompressCmd_Demo(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", dir)

	cmd := newCompressCmd()
	demoCmd, _, _ := cmd.Find([]string{"demo"})
	if err := demoCmd.RunE(demoCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestCompressCmd_Prune(t *testing.T) {
	t.Parallel()
	cmd := newCompressCmd()
	pruneCmd, _, _ := cmd.Find([]string{"prune"})
	if err := pruneCmd.RunE(pruneCmd, []string{"a\nb\nc\nd\ne\nf\ng\nh\ni\nj"}); err != nil {
		t.Fatal(err)
	}
}

func TestCompressCmd_ReportEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", dir)

	cmd := newCompressCmd()
	cmd.SetArgs([]string{"report"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestCompressCmd_Save(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", dir)

	cmd := newCompressCmd()
	saveCmd, _, _ := cmd.Find([]string{"save"})
	if err := saveCmd.RunE(saveCmd, []string{"test content to compress", "test"}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateCmd_EmptyDB(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newValidateCmd(&dbPath)
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected error for empty DB with no providers")
	}
}

func TestValidateCmd_NoProviders(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertAgent(&models.Agent{
		ID: "orphan-agent", Description: "Orphan", Mode: "auto", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newValidateCmd(&dbPath)
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected error with no providers")
	}
}

func TestSnapshotsCmd_Subcommands(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	cmd := newSnapshotsCmd(&dbPath)
	for _, name := range []string{"list", "show", "delete"} {
		sub, _, _ := cmd.Find([]string{name})
		if sub == nil {
			t.Errorf("expected subcommand %q", name)
		}
	}
}

func TestSnapshotsCmd_ListEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSnapshotsCmd(&dbPath)
	listCmd, _, _ := cmd.Find([]string{"list"})
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotsCmd_ListWithSnapshots(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.InsertSnapshot("abc123hash", "snapshot content"); err != nil {
		t.Fatal(err)
	}
	if err := d.InsertSnapshot("def456hash", "another snapshot"); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSnapshotsCmd(&dbPath)
	listCmd, _, _ := cmd.Find([]string{"list"})
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotsCmd_Show(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.InsertSnapshot("hash123", "show me this content"); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSnapshotsCmd(&dbPath)
	showCmd, _, _ := cmd.Find([]string{"show"})
	if err := showCmd.RunE(showCmd, []string{"1"}); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotsCmd_ShowInvalidID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSnapshotsCmd(&dbPath)
	showCmd, _, _ := cmd.Find([]string{"show"})
	if err := showCmd.RunE(showCmd, []string{"abc"}); err == nil {
		t.Fatal("expected error for non-numeric id")
	}
}

func TestSnapshotsCmd_ShowNonexistent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSnapshotsCmd(&dbPath)
	showCmd, _, _ := cmd.Find([]string{"show"})
	if err := showCmd.RunE(showCmd, []string{"999"}); err == nil {
		t.Fatal("expected error for nonexistent snapshot")
	}
}

func TestSnapshotsCmd_Delete(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.InsertSnapshot("delhash", "delete me"); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSnapshotsCmd(&dbPath)
	deleteCmd, _, _ := cmd.Find([]string{"delete"})
	if err := deleteCmd.RunE(deleteCmd, []string{"1"}); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotsCmd_DeleteInvalidID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSnapshotsCmd(&dbPath)
	deleteCmd, _, _ := cmd.Find([]string{"delete"})
	if err := deleteCmd.RunE(deleteCmd, []string{"abc"}); err == nil {
		t.Fatal("expected error for non-numeric id")
	}
}

func TestPrefsCmd_Subcommands(t *testing.T) {
	t.Parallel()
	cmd := newPreferencesCmd(nil)
	for _, name := range []string{"list", "get", "set", "delete"} {
		sub, _, _ := cmd.Find([]string{name})
		if sub == nil {
			t.Errorf("expected subcommand %q", name)
		}
	}
}

func TestPrefsCmd_ListEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newPreferencesCmd(&dbPath)
	listCmd, _, _ := cmd.Find([]string{"list"})
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestPrefsCmd_CRUD(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newPreferencesCmd(&dbPath)

	// Set
	setCmd, _, _ := cmd.Find([]string{"set"})
	if err := setCmd.RunE(setCmd, []string{"theme", "dark"}); err != nil {
		t.Fatal(err)
	}

	// Get
	getCmd, _, _ := cmd.Find([]string{"get"})
	if err := getCmd.RunE(getCmd, []string{"theme"}); err != nil {
		t.Fatal(err)
	}

	// Get nonexistent
	if err := getCmd.RunE(getCmd, []string{"nonexistent"}); err == nil {
		t.Fatal("expected error for nonexistent preference")
	}

	// List
	listCmd, _, _ := cmd.Find([]string{"list"})
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}

	// Delete
	deleteCmd, _, _ := cmd.Find([]string{"delete"})
	if err := deleteCmd.RunE(deleteCmd, []string{"theme"}); err != nil {
		t.Fatal(err)
	}

	// Delete nonexistent
	if err := deleteCmd.RunE(deleteCmd, []string{"theme"}); err == nil {
		t.Fatal("expected error deleting already-deleted preference")
	}
}

func TestPrefsCmd_SetMultiple(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newPreferencesCmd(&dbPath)
	setCmd, _, _ := cmd.Find([]string{"set"})
	for _, kv := range []struct{ k, v string }{
		{"editor", "helix"}, {"shell", "zsh"}, {"theme", "solarized"},
	} {
		if err := setCmd.RunE(setCmd, []string{kv.k, kv.v}); err != nil {
			t.Fatal(err)
		}
	}

	listCmd, _, _ := cmd.Find([]string{"list"})
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}
