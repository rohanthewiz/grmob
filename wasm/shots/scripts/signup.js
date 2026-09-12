// grmob-shot: {"app": "signup", "w": 414, "h": 640}
//
// docs/images/signup.png — a form complaining about one transposed
// character.
//
// The two password fields share a placeholder and are distinguished only
// by position, which is why they are addressed through byAttr rather than
// through typeInto: there is nothing to match on but order, and the form
// says so itself (see examples/signup/app_test.go's secondPasswordField).
//
// hunter2222 against hunter2221 is one character, and it is the smallest
// edit that makes the cross-field rule fire. A bigger difference would
// photograph the same message with a weaker claim behind it.
await typeInto("you@example.com", "ada@lovelace.dev");

const passwords = byAttr("data-listener_on-change").filter((f) => f.type === "password");
if (passwords.length !== 2) {
    throw new Error(`expected 2 password fields, found ${passwords.length}`);
}
await invoke(passwords[0], "data-listener_on-change", { value: "hunter2222" });
await invoke(passwords[1], "data-listener_on-change", { value: "hunter2221" });

await toggle("I accept the terms of service", true);

// The submit is what turns the explanations on. Under RevealOnBlur the
// confirmation field has not been left, so without this the form is in
// the state the picture is not of: filled, wrong, and silent.
await tap("Create account");
