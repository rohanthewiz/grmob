// grmob-shot: {"app": "mobileapp", "w": 414, "h": 560}
//
// docs/images/tabs-list.png — the Feed tab with one row selected.
//
// The tab is chosen by index because a tab strip is drawn by the renderer
// and the index is what the app's own callback carries; Feed is the third
// of four. The row is chosen by what it says, which is the identity a list
// row has — its position is a property of the scroll, not of the article.
await selectTab(2);
await tap("Article 3");
