import { App } from "../../assets/themes";
import { style } from "@vanilla-extract/css";

export const Main = style({
  background: App.Theme.color.background,
  display: "block",
  minWidth: App.Theme.size.appWidthMin,
  minHeight: "100vh",
  position: "relative",
  width: "100vw"
});