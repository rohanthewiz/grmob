// grmob-shot: {"app": "tutorial", "w": 414, "h": 800}
//
// docs/images/tutorial-lesson.png — lesson 1.1, scrolled to its demo.
//
// The scroll is the point of the shot. A lesson is a paragraph, a
// highlighted code block and a live TRY IT panel, and the top of the
// screen is a title bar — so a picture taken where the lesson opens shows
// the chrome and one paragraph. Scrolling to the code block puts the
// three things the shot is actually about in one frame, and is why the
// claim in internal/shotclaims names neither the lesson's own title nor
// its opening sentence: both are above the top of the picture.
//
// It scrolled to the opening sentence until the demo's toggles became
// comps.CheckboxRow. Those are full-width rows stacked in a column rather
// than three checkboxes squeezed into one row, which is taller, and from
// the opening sentence the profile card they recompose fell below the
// frame. A picture of the toggles without the card they drive shows the
// controls and not what they do, so the frame moved down one paragraph.
await tap("Hello, GrMob");
await scrollTo("type View interface");
