// grmob-shot: {"app": "todoapp", "w": 414, "h": 600}
//
// docs/images/todo.png — four tasks, two of them done.
//
// Typed and committed rather than seeded, which is what makes the footer's
// "2 items left" and the presence of the Clear completed button facts
// about the app rather than about this script: both are derived on every
// pass from the list, and neither can be posed.
//
// The titles are also in the claim (internal/shotclaims), because the
// picture shows them and a test that drove four different ones would be
// asserting a screen nobody can compare with the image.
for (const title of ["Buy oat milk", "Read the GrMob docs",
                     "Ship the beta", "Call Ada"]) {
    await typeInto("What needs doing?", title);
    await tap("Add");
}
await toggle("Read the GrMob docs", true);
await toggle("Call Ada", true);
