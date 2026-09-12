// grmob-shot: {"page": "hero.html", "clip": "#hero"}
//
// docs/images/hero.png — the three app shots side by side.
//
// A picture of pictures, and therefore the one shot with no app in it:
// hero.html arranges the finished PNGs and the camera photographs that.
//
// # Why a browser and not an image library
//
// Compositing three PNGs is a solved problem with a dependency attached,
// and this repository has a Chrome already running for exactly this file.
// Laying them out in HTML costs one page, no module, and gives the gaps
// and the ground the same declarative form as everything else here.
//
// It must run AFTER the three it is made of. shoot.sh with no arguments
// orders them; running hero on its own composites whatever is currently
// in docs/images, which is the right behaviour and worth knowing about.
