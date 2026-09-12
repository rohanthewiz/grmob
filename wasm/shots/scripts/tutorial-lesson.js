// grmob-shot: {"app": "tutorial", "w": 414, "h": 800}
//
// docs/images/tutorial-lesson.png — lesson 1.1, scrolled to its demo.
//
// The scroll is the point of the shot. A lesson is a paragraph, a
// highlighted code block and a live TRY IT panel, and the top of the
// screen is a title bar — so a picture taken where the lesson opens shows
// the chrome and one paragraph. Scrolling to the opening sentence puts
// the three things the shot is actually about in one frame, and is why
// the claim in internal/shotclaims does not name the lesson's own title:
// it is above the top of the picture.
await tap("Hello, GrMob");
await scrollTo("A GrMob screen is not a template");
