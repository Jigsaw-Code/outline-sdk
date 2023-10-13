import { style } from "@vanilla-extract/css";
import { App } from "../../../../assets/themes";

export const Main = style({
  background: App.Theme.color.backgroundBrand,
  display: "flex",
  justifyContent: "center",
  padding: App.Theme.size.gap,
  position: "sticky",
  top: 0,
  width: "100%",
});

export const Text = style({
  color: App.Theme.color.textBrand,
  fontFamily: App.Theme.font.sansSerif,
  fontSize: App.Theme.size.fontLarge,
  maxWidth: App.Theme.size.appWidthMax,
  textAlign: "center",
});
