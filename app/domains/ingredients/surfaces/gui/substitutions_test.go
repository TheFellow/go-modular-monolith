//nolint:paralleltest // Fyne's headless application and driver state are process-global.
package gui

import (
	inventorymodels "github.com/TheFellow/go-modular-monolith/app/domains/inventory/models"
	"github.com/TheFellow/go-modular-monolith/app/kernel/currency"
	"github.com/TheFellow/go-modular-monolith/app/kernel/measurement"
	"github.com/TheFellow/go-modular-monolith/app/kernel/money"
	"strings"
	"testing"

	frameworktest "fyne.io/fyne/v2/test"
	application "github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/app/domains/ingredients/models"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/testutil/fynetest"
	toolkit "github.com/TheFellow/go-modular-monolith/pkg/toolkits/gui"
)

func TestSubstitutionWidgetsCreateReviseDisableAndRejectStaleRevision(t *testing.T) {
	gui := frameworktest.NewApp()
	t.Cleanup(gui.Quit)
	f, original, substitute := ingredientFixture(t)
	p, dialogs := newTestPresenter(f.App, toolkit.InlineExecutor{})
	v := NewView(p)
	driver := fynetest.NewDriver(t, v.Content())
	p.Load()
	p.Select(original.ID)
	driver.Tap(ControlSubstitutions)
	driver.Type(ControlRuleSubstitute, substitute.ID.String())
	driver.Type(ControlRuleRatio, "1.5")
	driver.Type(ControlRuleNotes, "Fresh stock alternative")
	v.rules.quality.SetSelected("similar")
	driver.Tap(ControlRuleSave)
	rules, err := f.Ingredients.SubstitutionRules(f.OwnerContext(), original.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(rules), 1)
	testutil.Equals(t, rules[0].Ratio, 1.5)
	testutil.Equals(t, rules[0].QualityImpact, models.QualitySimilar)
	v.rules.rows.Select(0)
	testutil.Equals(t, v.rules.substitute.Disabled(), true)
	stale := rules[0]
	latest := rules[0]
	latest.Notes = "Concurrent change"
	saved, err := f.Ingredients.SetSubstitution(f.OwnerContext(), &latest)
	testutil.Ok(t, err)
	driver.Type(ControlRuleNotes, "Stale change")
	driver.Tap(ControlRuleSave)
	testutil.ErrorIsConflict(t, p.Snapshot().Err)
	testutil.Equals(t, p.Snapshot().RuleForm.Revision, stale.Revision)
	testutil.Equals(t, p.Snapshot().RuleForm.Notes, "Stale change")
	driver.Tap(ControlRuleRefresh)
	// Reselect the refreshed record deliberately before retrying.
	p.SelectSubstitution(substitute.ID)
	confirmations := dialogs.Confirmations()
	testutil.Equals(t, len(confirmations), 1)
	confirmations[0].Respond(false)
	testutil.Equals(t, p.Snapshot().RuleForm.Notes, "Stale change")
	testutil.Equals(t, p.Snapshot().RuleForm.Revision, stale.Revision)
	p.SelectSubstitution(substitute.ID)
	confirmations = dialogs.Confirmations()
	testutil.Equals(t, len(confirmations), 2)
	confirmations[1].Respond(true)
	testutil.Equals(t, p.Snapshot().RuleForm.Revision, saved.Revision)
	frameworktest.Tap(v.rules.disabled)
	driver.Tap(ControlRuleSave)
	rules, err = f.Ingredients.SubstitutionRules(f.OwnerContext(), original.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, rules[0].Disabled, true)
	testutil.Equals(t, len(p.Snapshot().Rules), 1)
	p.SelectSubstitution(substitute.ID)
	frameworktest.Tap(v.rules.disabled)
	driver.Tap(ControlRuleSave)
	enabled, err := f.Ingredients.SubstitutionsFor(f.OwnerContext(), original.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, len(enabled), 1)
}

func TestReadOnlySubstitutionWidgetsAndMutationGuard(t *testing.T) {
	gui := frameworktest.NewApp()
	t.Cleanup(gui.Quit)
	f, original, substitute := ingredientFixture(t)
	_, err := f.Ingredients.SetSubstitution(f.OwnerContext(), &models.SubstitutionRule{IngredientID: original.ID, SubstituteID: substitute.ID, Ratio: 1, QualityImpact: models.QualityEquivalent})
	testutil.Ok(t, err)
	session := application.NewSession(f.ActorContext("bartender"), f.App.App)
	p, _ := newTestPresenter(session, toolkit.InlineExecutor{})
	v := NewView(p)
	p.Load()
	p.Select(original.ID)
	fynetest.NewDriver(t, v.Content()).Tap(ControlSubstitutions)
	testutil.Equals(t, len(p.Snapshot().Rules), 1)
	testutil.Equals(t, v.rules.create.Visible(), false)
	testutil.Equals(t, v.rules.save.Visible(), false)
	testutil.Equals(t, v.rules.ratio.Disabled(), true)
	testutil.Equals(t, p.SubmitSubstitution(), false)
}

func TestRetirementWidgetsWithdrawStockWithReason(t *testing.T) {
	gui := frameworktest.NewApp()
	t.Cleanup(gui.Quit)
	f, original, _ := ingredientFixture(t)
	testutil.SetInventory(t, f, inventorymodels.Update{CostPerUnit: money.NewPriceFromCents(100, currency.USD), IngredientID: original.ID, Amount: measurement.MustAmount(10, original.Unit)})
	p, dialogs := newTestPresenter(f.App, toolkit.InlineExecutor{})
	v := NewView(p)
	p.Load()
	p.Select(original.ID)
	frameworktest.Tap(v.withdraw)
	fynetest.NewDriver(t, v.Content()).Type(ControlDelete+".reason", "Contamination")
	fynetest.NewDriver(t, v.Content()).Tap(ControlDelete)
	confirmations := dialogs.Confirmations()
	testutil.Equals(t, len(confirmations), 1)
	testutil.Equals(t, strings.Contains(confirmations[0].Message, "quarantined"), true)
	confirmations[0].Respond(true)
	stock, err := f.Inventory.Get(f.OwnerContext(), original.ID)
	testutil.Ok(t, err)
	testutil.Equals(t, stock.Status, inventorymodels.StatusQuarantined)
	testutil.Equals(t, stock.Reason, "Contamination")
}
