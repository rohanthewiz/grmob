package core

import "testing"

// Roles(), pinned to the const blocks in role.go that it restates. See
// enum_pin_test.go for the parse and why it is one, and role.go for what the
// list obliges each renderer to do.
//
// This is the pin the native coverage checks in mobile/verify hang off: they
// hold each renderer's dispatch against Roles(), so if Roles() could quietly
// stop listing a constant, both of them would quietly stop requiring an arm
// for it.

// The census is checked against the declarations *plus the zero value*,
// because RoleNone is a declared Role that Roles() deliberately omits — it is
// the absence of a role, and no renderer has an arm for "unset". Adding it
// back here is what lets the check stay exact in both directions: every
// declared constant is accounted for, and nothing is listed that core cannot
// produce.
func TestRolesMatchTheDeclaredConstants(t *testing.T) {
	requireExactEnum(t, "role.go", "Role", "Roles() plus RoleNone", append(Roles(), RoleNone))
}

// The other half of that arrangement: RoleNone must stay out of the census and
// must stay the empty string. A RoleNone that crept into Roles() would oblige
// every renderer to grow an arm for the value that means "no role", and a
// RoleNone with a spelling would stop being the Style field's zero value —
// every node in the tree would start carrying it.
func TestRolesOmitsTheZeroValue(t *testing.T) {
	if RoleNone != "" {
		t.Errorf("RoleNone = %q, want the empty string so an unset Style.AccessibilityRole is it", RoleNone)
	}
	for _, role := range Roles() {
		if role == RoleNone {
			t.Error("Roles() lists RoleNone; the census is the roles that do something")
		}
	}
}
