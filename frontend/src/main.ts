import { mount, type Component } from "svelte";
import "./app.css";
import Widget from "./views/Widget.svelte";
import Popover from "./views/Popover.svelte";
import Settings from "./views/Settings.svelte";
import { app } from "./lib/state.svelte";

// Every window loads the same bundle; the Go side picks the view with "/?view=<name>"
// (see internal/app/contract.go).
const views: Record<string, Component> = { widget: Widget, popover: Popover, settings: Settings };
const name = new URLSearchParams(location.search).get("view") ?? "settings";
document.documentElement.dataset.view = name;

app.start();
mount(views[name] ?? Settings, { target: document.getElementById("app")! });
