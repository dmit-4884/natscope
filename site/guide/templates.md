---
title: Templates
description: Save a subject, message type, payload and headers as a reusable publish preset.
---

# Templates

A template is a saved publish draft: subject, message type, JSON body and headers in one preset. Use
them for the messages you send over and over while testing a service.

<Video src="/media/templates.mp4" poster="/media/templates.jpg" caption="Saving a publish draft as a template and loading it back." />

## Save a template

From the **Publish** tab of any stream, fill in the form and click **Save as template**. Give it a name
and it lands in the template list.

## Use a template

The **Publish** tab has a **Templates** dropdown showing how many you have. Pick one and it loads its
subject, payload and headers into the form. A toast offers **Undo** if you picked the wrong one.

The dropdown also links to **Manage all in Settings →**.

## Manage templates

**Settings → Templates** lists them all with **Name**, **Subject**, **Message type**, **Headers** and
**Updated** columns.

- **New template** creates one from scratch
- Row actions: edit, duplicate, delete
- **Import** and **Export** move templates between workspaces as JSON
- **Copy** puts the selection on the clipboard
- Select several rows for **Delete selected**, or wipe the list with **Clear all**

<Shot src="/media/templates-list.png" alt="Template list" />

Templates travel in a [workspace export](/guide/workspace) as their own section.
