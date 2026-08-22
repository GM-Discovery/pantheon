(() => {
  "use strict";
  const state = { inventory: null, work: null, config: null, rumor: null, mail: null, aphrodite: null, echo: null, fame: null };
  const $ = (id) => document.getElementById(id);
  const escapeHTML = (value) => String(value ?? "").replace(/[&<>"']/g, (c) => ({"&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;","'":"&#39;"}[c]));
  const json = async (url, options = {}) => { const response = await fetch(url, { credentials: "same-origin", ...options }); let body = {}; try { body = await response.json(); } catch (_) {} if (!response.ok) throw new Error(body.error || `Request failed (${response.status})`); return body; };
  const post = (url, body) => json(url, { method: "POST", headers: { "Content-Type": "application/json" }, body: body ? JSON.stringify(body) : "{}" });
  const showNotice = (message, error = false) => { const notice = $("notice"); notice.textContent = message; notice.hidden = !message; notice.classList.toggle("error", error); };
  const stateChip = (value) => `<span class="state-chip ${escapeHTML(String(value || "").toLowerCase().replace(/[^a-z]+/g, "-"))}">${escapeHTML(value || "unknown")}</span>`;
  const mapEntries = (value) => Object.entries(value || {}).sort((a, b) => a[0].localeCompare(b[0]));
  const sumMap = (value) => mapEntries(value).reduce((sum, item) => sum + (typeof item[1] === "number" ? item[1] : 0), 0);
  const countStates = (value, states) => states.reduce((sum, state) => sum + (Number(value?.[state]) || 0), 0);
  const renderMetrics = (target, values) => { $(target).innerHTML = Object.entries(values).map(([label, value]) => `<div class="metric"><strong>${escapeHTML(value)}</strong><span>${escapeHTML(label)}</span></div>`).join(""); };
  const renderWorkBoard = (work, workSnapshot) => {
    const future = work.future?.state_counts || {};
    const janus = work.janus?.state_counts || {};
    const mercury = work.mercury?.state_counts || {};
    const failures = work.failures || {};
    const liveCounts = workSnapshot?.counts || {};
    const items = [
      ["active", "Active work", liveCounts.active ?? (countStates(mercury, ["pending", "sealed", "ready", "dispatched", "running"]) + (Number(work.apollo?.pending_orders) || 0)), "Mercury or Apollo currently holds work", false],
      ["waiting", "Waiting work", liveCounts.waiting ?? (countStates(mercury, ["waiting_paused"]) + countStates(future, ["awaiting", "temporally_blocked"])), "paused, awaiting, or temporal", false],
      ["needs_grant", "Needs Grant", liveCounts.needs_grant ?? countStates(future, ["needs_authority", "referred_prometheus"]), "authority gap to resolve", (liveCounts.needs_grant ?? countStates(future, ["needs_authority", "referred_prometheus"])) > 0],
      ["blocked", "Blocked work", liveCounts.blocked ?? (countStates(future, ["adapter_unavailable", "malformed"]) + countStates(janus, ["blocked_error"]) + countStates(mercury, ["blocked_unavailable"])), "known boundary or failure", true],
      ["needs_attention", "Needs decision", liveCounts.needs_attention ?? countStates(mercury, ["needs_ruling", "needs_interpretation"]), "a bounded human ruling is required", (liveCounts.needs_attention ?? countStates(mercury, ["needs_ruling", "needs_interpretation"])) > 0],
      ["recent", "Recent completion", Number(mercury.completed) || 0, "completed Mercury orders", false],
      ["upcoming", "Upcoming known", liveCounts.upcoming ?? (countStates(future, ["awaiting"]) + countStates(janus, ["scheduled", "ready"])), "declared Future / Janus work", false],
    ];
    $("work-summary").innerHTML = items.map(([category, label, value, detail, priority]) => `<button type="button" class="work-item ${priority ? "priority" : ""}" data-work-category="${category}" aria-label="View ${escapeHTML(label)} details"><strong>${escapeHTML(value)}</strong><h3>${escapeHTML(label)}</h3><p>${escapeHTML(detail)}</p><span class="work-item-action">View records and next step →</span></button>`).join("");
    $("work-summary-note").textContent = `${workSnapshot ? "Identified live work" : "Summary counts"} · ${Number(failures.failed_or_blocked) || 0} failed or blocked audit events are shown separately as a diagnostic signal.`;
  };

  function selectView(view) {
    document.querySelectorAll(".view").forEach((panel) => panel.classList.toggle("active", panel.dataset.panel === view));
    document.querySelectorAll(".nav-item").forEach((item) => item.classList.toggle("active", item.dataset.view === view));
    if (view === "notifications") loadNotifications();
    if (view === "configuration") loadConfiguration();
    if (view === "bifrost") loadCapabilities();
    if (view === "rumor") loadRumor();
    if (view === "mail") loadMail();
    if (view === "zodiac") loadZodiac();
    if (view === "aphrodite") loadAphrodite();
    if (view === "echo") loadEcho();
    if (view === "fame") loadFame();
    if (view === "execution") loadExecutionWork($("execution-filter")?.value || "all");
  }
  document.querySelectorAll("[data-view]").forEach((item) => item.addEventListener("click", () => selectView(item.dataset.view)));
  document.addEventListener("click", (event) => {
    const workTrigger = event.target.closest("[data-work-category]");
    if (workTrigger) { loadWorkContext(workTrigger.dataset.workCategory); return; }
    const clearSegment = event.target.closest("[data-echo-clear-segment]");
    if (clearSegment) { clearEchoSegment(clearSegment.dataset.echoClearSegment); return; }
    const playSegment = event.target.closest("[data-echo-play-segment]");
    if (playSegment) { playEchoSegment(playSegment.dataset.echoPlaySegment, playSegment.dataset.echoPlayStart, playSegment.dataset.echoPlayEnd); return; }
    const editAssigned = event.target.closest("[data-echo-edit-assigned-segment]");
    if (editAssigned) {
      const segment = (state.echo?.segments || []).find((item) => item.segment_id === editAssigned.dataset.echoEditAssignedSegment);
      if (segment) renderEchoSpeakerDetail(segment.anonymous_speaker, segment.segment_id, true);
      return;
    }
    const renameVoice = event.target.closest("[data-echo-voice-rename]");
    if (renameVoice) { renameEchoVoice(renameVoice.dataset.echoVoiceRename); return; }
    const archiveVoice = event.target.closest("[data-echo-voice-archive]");
    if (archiveVoice) { archiveEchoVoice(archiveVoice.dataset.echoVoiceArchive); return; }
    const approveSpeaker = event.target.closest("[data-echo-approve-speaker]");
    if (approveSpeaker) { approveEchoSpeaker(approveSpeaker.dataset.echoApproveSpeaker, approveSpeaker.dataset.echoApproveSegment); return; }
    const speakerTrigger = event.target.closest("[data-echo-speaker]");
    if (speakerTrigger) { renderEchoSpeakerDetail(speakerTrigger.dataset.echoSpeaker); return; }
    const viewTrigger = event.target.closest("[data-view-link]");
    if (viewTrigger) selectView(viewTrigger.dataset.viewLink);
    if (event.target.closest("#work-context-close")) $("work-context").hidden = true;
    if (event.target.closest("#echo-speaker-detail-close")) $("echo-speaker-detail").hidden = true;
  });
  document.addEventListener("submit", (event) => {
    const form = event.target.closest("[data-echo-segment-form]");
    if (form) { event.preventDefault(); saveEchoSegmentIdentity(form); }
  });
  document.addEventListener("change", (event) => {
    const select = event.target.closest("[data-echo-profile-for]");
    if (!select) return;
    const input = select.closest("[data-echo-segment-form]")?.querySelector("[data-echo-name-for]");
    if (input && !input.value) input.value = select.options[select.selectedIndex]?.textContent?.split(" · ")[0] || "";
  });

  function renderInventory() {
    const inventory = state.inventory; if (!inventory) return;
    const paused = Boolean(inventory.runtime?.paused);
    $("runtime-state").textContent = paused ? "paused" : "online"; $("runtime-state").className = `status-dot ${paused ? "paused" : "ready"}`;
    $("prometheus-state").outerHTML = `<span id="prometheus-state" class="state-chip ${paused ? "paused" : "enabled"}">${paused ? "paused" : "running"}</span>`;
    $("pause-button").disabled = paused; $("resume-button").disabled = !paused;
    $("freshness").textContent = `Live state · ${new Date(inventory.generated_at).toLocaleTimeString()}`;
    const w = inventory.work || {};
    renderWorkBoard(w, state.work);
    renderMetrics("intake-summary", { "present occurrences": w.intake?.present_occurrences ?? 0, "failed scans": w.intake?.failed_scans ?? 0, "chronicles waiting": w.intake?.pending_chronicle_writes ?? 0 });
    $("pulse-state").outerHTML = `<span id="pulse-state" class="state-chip ${paused ? "paused" : "ready"}">${paused ? "paused" : "running"}</span>`;
    $("pulse-summary").innerHTML = [["Intake", `${w.intake?.present_occurrences ?? 0} present occurrences`], ["Rumor", `${sumMap(w.rumor?.state_counts)} tracked sources`], ["Failure signal", `${w.failures?.failed_or_blocked ?? 0} failed or blocked`], ["Latest Intake activity", w.intake?.latest_activity_at || "none recorded"]].map(([k,v]) => `<div class="detail-row"><span>${escapeHTML(k)}</span><strong>${escapeHTML(v)}</strong></div>`).join("");
    $("cells").innerHTML = (inventory.components || []).map((component) => `<article class="cell ${component.state === "deferred" ? "deferred" : ""}"><h3>${escapeHTML(component.label)}</h3><p>${escapeHTML(component.role)}</p>${stateChip(component.state)} <small>${escapeHTML(component.kind)}</small></article>`).join("");
    $("cell-summary").textContent = `${(inventory.components || []).length} visible cells · updated from live inventory`;
    $("apollo-state").outerHTML = `<span id="apollo-state" class="state-chip">${escapeHTML(w.apollo?.pending_orders ?? 0)} pending · ${escapeHTML(w.apollo?.completed_reports ?? 0)} reports</span>`;
    $("apollo-summary").innerHTML = [["Model", inventory.models?.apollo?.model || "not configured"], ["Extraction model", inventory.models?.apollo?.extraction_model || "falls back to Apollo"], ["Citations", w.apollo?.citation_required ? "required" : "optional"], ["External research", w.apollo?.external_research_enabled ? "enabled by config" : "disabled"]].map(([k,v]) => `<div class="detail-row"><span>${escapeHTML(k)}</span><strong>${escapeHTML(v)}</strong></div>`).join("");
    $("system-details").innerHTML = [["Contract version", inventory.contract_version], ["Generated", inventory.generated_at], ["Listen address", inventory.runtime?.listen_address], ["Audit events", inventory.runtime?.audit_events], ["Database", inventory.runtime?.database_path], ["Secrets", inventory.security?.secrets]].map(([k,v]) => `<div class="detail-row"><span>${escapeHTML(k)}</span><strong>${escapeHTML(v)}</strong></div>`).join("");
    $("system-components").innerHTML = (inventory.components || []).map((c) => `<article class="component"><div class="component-head"><h3>${escapeHTML(c.label)}</h3>${stateChip(c.state)}</div><p>${escapeHTML(c.role)}${c.inspection_route ? ` · <code>${escapeHTML(c.inspection_route)}</code>` : ""}</p></article>`).join("");
    $("models-list").innerHTML = [["Apollo", inventory.models?.apollo, "Local cited reasoning"], ["Archimedes", inventory.models?.archimedes, "Frontier boundary; deferred"]].map(([name, model, description]) => `<article class="model-card"><h2>${escapeHTML(name)} ${stateChip(model?.state)}</h2><p>${escapeHTML(description)}</p><p>Model: <code>${escapeHTML(model?.model || "not configured")}</code></p>${name === "Apollo" ? `<p>Extraction model: <code>${escapeHTML(model?.extraction_model || "falls back to Apollo")}</code></p>` : ""}<p>Local-only by default: <strong>${escapeHTML(model?.local_only_by_default ?? !model?.enabled)}</strong></p></article>`).join("");
    renderMailHome();
  }

  async function refreshState() { try { const results = await Promise.allSettled([json("/v1/system/inventory"), json("/v1/gui/work"), json("/v1/gui/mail")]); if (results[0].status === "rejected") throw results[0].reason; state.inventory = results[0].value; if (results[1].status === "fulfilled") state.work = results[1].value; else { state.work = null; console.warn("[Panopticon] Work context unavailable", results[1].reason); } if (results[2].status === "fulfilled") state.mail = results[2].value; else { state.mail = null; console.warn("[Panopticon] Mail state unavailable", results[2].reason); } renderInventory(); } catch (error) { $("runtime-state").textContent = "unavailable"; $("runtime-state").className = "status-dot paused"; showNotice("Live Pantheon state unavailable. Retry shortly.", true); console.warn("[Panopticon] Live state unavailable", error); } }
  function renderAphrodite(body) {
    state.aphrodite = body;
    const project = body.active_project || body.projects?.[0] || {};
    $("aphrodite-project-name").textContent = project.name || "Project state";
    $("aphrodite-project-purpose").textContent = project.purpose || "No active project is recorded.";
    $("aphrodite-project-state").outerHTML = `<span id="aphrodite-project-state" class="state-chip ${escapeHTML(project.state || "unknown")}">${escapeHTML(project.state || "unknown")}</span>`;
    renderMetrics("aphrodite-summary", { "projects": (body.projects || []).length, "Fame fixtures": (body.fame_fixtures || []).length, "research requests": (body.research_requests || []).length, "candidate artifacts": (body.artifacts || []).length });
    $("aphrodite-project-meta").innerHTML = [["Phase", project.phase || "not started"], ["Workspace", project.workspace_path || body.workspace_root || "not available"], ["Last result", project.last_result || "none recorded"], ["Approved inspection routes", (body.approved_inspection_routes || []).length]].map(([k, v]) => `<div class="detail-row"><span>${escapeHTML(k)}</span><strong>${escapeHTML(v)}</strong></div>`).join("");
    const fixtures = body.fame_fixtures || [];
    $("aphrodite-fixture-count").textContent = `${fixtures.length} fixture${fixtures.length === 1 ? "" : "s"}`;
    $("aphrodite-fixtures").innerHTML = fixtures.map((item) => `<article class="work-detail"><div class="work-detail-head"><strong>${escapeHTML(item.title)}</strong>${stateChip(item.source_kind)}</div><p>${escapeHTML(item.transcript_path)}</p><div class="zodiac-meta">sha256 ${escapeHTML(item.sha256)}${item.source_id ? ` · Rumor ${escapeHTML(item.source_id)}` : " · corpus case"}</div></article>`).join("") || `<p class="muted">No Fame fixtures recorded.</p>`;
    const requests = body.research_requests || [];
    $("aphrodite-research-count").textContent = `${requests.length} request${requests.length === 1 ? "" : "s"}`;
    $("aphrodite-research").innerHTML = requests.map((item) => `<article class="work-detail"><div class="work-detail-head"><strong>${escapeHTML(item.question)}</strong>${stateChip(item.status)}</div><p>${escapeHTML(item.rationale || "No rationale recorded.")}</p><div class="zodiac-meta">${item.surfaced_in_brief ? "visible in Situation Brief" : "not surfaced in Situation Brief"} · ${escapeHTML(item.artifact_path || "no artifact path")}</div></article>`).join("") || `<p class="muted">No Apollo research requests recorded.</p>`;
    const artifacts = body.artifacts || [];
    $("aphrodite-artifact-count").textContent = `${artifacts.length} artifact${artifacts.length === 1 ? "" : "s"}`;
    $("aphrodite-artifacts").innerHTML = artifacts.map((item) => `<article class="work-detail"><div class="work-detail-head"><strong>${escapeHTML(item.title)}</strong>${stateChip(item.kind)}</div><p>${escapeHTML(item.path)}</p><div class="zodiac-meta">${escapeHTML(item.status)} · sha256 ${escapeHTML(item.sha256)}</div></article>`).join("") || `<p class="muted">No candidate artifacts recorded.</p>`;
    $("aphrodite-boundaries").innerHTML = (body.boundaries || []).map((item) => `<li>${escapeHTML(item)}</li>`).join("");
  }
  async function loadAphrodite() { try { renderAphrodite(await json("/v1/gui/aphrodite")); } catch (error) { $("aphrodite-project-purpose").textContent = "Aphrodite state is unavailable; no project action was taken."; showNotice(error.message || "Aphrodite state unavailable.", true); } }
  function renderEcho(body) {
    state.echo = body;
    const run = body.latest_run || {};
    const profiles = Array.isArray(body.profiles) ? body.profiles : [];
    const speakers = Array.isArray(body.anonymous_speakers) ? body.anonymous_speakers : [];
    const runs = Array.isArray(body.recent_runs) ? body.recent_runs : [];
    $("echo-state").outerHTML = `<span id="echo-state" class="state-chip ${body.enabled ? "enabled" : "disabled"}">${body.enabled ? "local" : "disabled"}</span>`;
    renderMetrics("echo-summary", { "voice profiles": profiles.length, "unassigned speakers": speakers.filter((speaker) => (body.segments || []).some((segment) => segment.anonymous_speaker === speaker && !segment.voice_id)).length, "segments in latest run": run.segment_count || 0, "confirmed mappings": (body.segments || []).filter((s) => s.mapping_state === "confirmed").length });
    $("echo-model").textContent = `${body.model || "not configured"} · ${body.local_only ? "local only" : "external boundary"}`;
    $("echo-run-select").innerHTML = runs.map((item) => `<option value="${escapeHTML(item.run_id)}">${escapeHTML(item.run_id)} · ${escapeHTML(item.speaker_count)} speakers · ${escapeHTML(item.source_path)}</option>`).join("");
    if (body.selected_run_id) $("echo-run-select").value = body.selected_run_id;
    $("echo-run").innerHTML = run.run_id ? `<article class="work-detail"><div class="work-detail-head"><strong>${escapeHTML(run.run_id)}</strong>${stateChip(run.status)}</div><p>${escapeHTML(run.source_path)}</p><div class="zodiac-meta">${escapeHTML(run.segment_count)} segments · ${escapeHTML(run.speaker_count)} anonymous speakers · diarization ${escapeHTML(run.diarization_path)}</div></article>` : `<p class="muted">No diarization run has been imported yet.</p>`;
    $("echo-profiles").innerHTML = profiles.map((p) => `<article class="work-detail echo-profile-card"><div class="work-detail-head"><div><input class="echo-profile-name" data-echo-voice-name="${escapeHTML(p.voice_id)}" value="${escapeHTML(p.name)}" aria-label="Voice profile name"></div>${stateChip(p.state)}</div><div class="button-row"><button type="button" class="button quiet" data-echo-voice-rename="${escapeHTML(p.voice_id)}">Save name</button>${p.voice_id !== "me" ? `<button type="button" class="button quiet" data-echo-voice-archive="${escapeHTML(p.voice_id)}">Remove</button>` : ""}<span class="muted">${p.sample_count ? `${escapeHTML(p.sample_count)} local sample reference${p.sample_count === 1 ? "" : "s"}` : "No sample yet"}</span></div><div class="zodiac-meta">voice id ${escapeHTML(p.voice_id)} · local only ${escapeHTML(p.local_only)}</div></article>`).join("") || `<p class="muted">No active voice profiles registered.</p>`;
    $("echo-assigned").innerHTML = profiles.map((profile) => {
      const assigned = (body.segments || []).filter((segment) => segment.voice_id === profile.voice_id);
      if (!assigned.length) return "";
      const substantial = assigned.filter((segment) => echoSegmentDuration(segment) >= 1);
      const visible = substantial.length ? substantial : assigned;
      const rows = visible.slice(0, 10).map((segment) => "<li><span><b>" + escapeHTML(formatEchoSeconds(segment.start_seconds)) + "</b> · " + escapeHTML(segment.transcript_snippet || "Transcript snippet unavailable") + "</span><button type=\"button\" class=\"text-button\" data-echo-edit-assigned-segment=\"" + escapeHTML(segment.segment_id) + "\">Edit assignment</button></li>").join("");
      const note = substantial.length < assigned.length ? "<p class=\"muted\">Short derived fragments are omitted from this profile view.</p>" : "";
      return "<article class=\"work-detail echo-assigned-card\"><div class=\"work-detail-head\"><strong>" + escapeHTML(profile.name) + "</strong>" + stateChip(assigned.length + " assigned") + "</div><ul>" + rows + "</ul>" + note + (visible.length > 10 ? "<p class=\"muted\">Showing 10 of " + visible.length + " visible assigned snippets.</p>" : "") + "</article>";
    }).join("") || "<p class=\"muted\">No snippets have been assigned to a voice profile yet.</p>";
    $("echo-speakers").innerHTML = speakers.map((speaker) => {
      const allSpeakerSegments = (body.segments || []).filter((segment) => segment.anonymous_speaker === speaker);
      const segments = allSpeakerSegments.filter((segment) => !segment.voice_id);
      if (!segments.length) return "";
      const first = segments[0];
      const firstTime = first ? formatEchoSeconds(first.start_seconds) : "not in the visible segment window";
      const mapped = allSpeakerSegments.filter((segment) => segment.voice_id).map((segment) => profiles.find((profile) => profile.voice_id === segment.voice_id)?.name || segment.voice_id);
      const uniqueMapped = [...new Set(mapped)];
      return `<button type="button" class="work-detail echo-speaker-card" data-echo-speaker="${escapeHTML(speaker)}"><div class="work-detail-head"><strong>${escapeHTML(speaker)}</strong>${stateChip(mapped.length ? "review assignments" : "anonymous")}</div><p>${mapped.length ? `Assigned in ${mapped.length} visible segment${mapped.length === 1 ? "" : "s"}: ${escapeHTML(uniqueMapped.join(", "))}.` : "No identity assignment yet."}</p><div class="zodiac-meta">${segments.length} visible segment${segments.length === 1 ? "" : "s"} · first visible timestamp ${escapeHTML(firstTime)} · review snippets and approve deliberately</div></button>`;
    }).join("") || `<p class="muted">No anonymous speaker labels recorded.</p>`;
    $("echo-boundaries").innerHTML = (body.boundaries || []).map((item) => `<li>${escapeHTML(item)}</li>`).join("");
  }
  const formatEchoSeconds = (value) => { const seconds = Math.max(0, Math.floor(Number(value) || 0)); const hours = Math.floor(seconds / 3600); const minutes = Math.floor((seconds % 3600) / 60); const remainder = seconds % 60; return `${String(hours).padStart(2, "0")}:${String(minutes).padStart(2, "0")}:${String(remainder).padStart(2, "0")}`; };
  const echoSegmentDuration = (segment) => Math.max(0, Number(segment?.end_seconds || 0) - Number(segment?.start_seconds || 0));
  function buildEchoReviewWindows(segments, includeAssigned = false) {
    const sorted = [...segments].sort((a, b) => Number(a.start_seconds) - Number(b.start_seconds));
    const windows = [];
    for (const segment of sorted) {
      const last = windows[windows.length - 1];
      const forceSingle = includeAssigned && Boolean(segment.voice_id);
      const sameAssignment = last && (last.voice_id || "") === (segment.voice_id || "");
      const gap = last ? Number(segment.start_seconds) - last.end_seconds : Infinity;
      const span = last ? Number(segment.end_seconds) - last.start_seconds : Infinity;
      if (!last || forceSingle || !sameAssignment || gap > 1.25 || span > 10) {
        windows.push({
          id: segment.segment_id,
          start_seconds: Number(segment.start_seconds),
          end_seconds: Number(segment.end_seconds),
          voice_id: segment.voice_id || "",
          segments: [segment],
        });
        continue;
      }
      last.segments.push(segment);
      last.end_seconds = Math.max(last.end_seconds, Number(segment.end_seconds));
    }
    return windows.map((window) => ({
      ...window,
      transcript: [...new Set(window.segments.map((segment) => segment.transcript_snippet).filter(Boolean))].join(" "),
      duration: Math.max(0, window.end_seconds - window.start_seconds),
    }));
  }
  function renderEchoSpeakerDetail(speaker, selectedSegmentID = "", includeAssigned = false) {
    const body = state.echo || {};
    const run = body.latest_run || {};
    const segments = (body.segments || []).filter((segment) => segment.anonymous_speaker === speaker && (includeAssigned || !segment.voice_id));
    const allWindows = buildEchoReviewWindows(segments, includeAssigned);
    $("echo-speaker-detail-intro").textContent = "Each review window below has its own audio, transcript, and assignment controls. Bulk assignment, when used, is limited to this anonymous speaker in this selected run.";
    const usefulWindows = allWindows.filter((window) => window.duration >= 1 || allWindows.every((candidate) => candidate.duration < 1));
    const reviewWindows = usefulWindows.slice(0, 10);
    const hiddenShortWindows = usefulWindows.filter((window) => window.duration < 1).length;
    const omittedWindows = Math.max(0, usefulWindows.length - reviewWindows.length);
    const source = run.source_path || "the latest Echo run";
    const selectedWindow = null;
    $("echo-speaker-detail").hidden = false;
    $("echo-speaker-detail-title").textContent = speaker;
    $("echo-speaker-detail-intro").textContent = `This is ${speaker} in run ${run.run_id || "without an imported run"}. Select one transcript snippet, listen to that time range, and assign only that segment. Later segments remain independent until confirmed.`;
    const confirmedSample = (body.segments || []).find((segment) => segment.anonymous_speaker === speaker && segment.voice_id);
    $("echo-speaker-sample-note").textContent = "Choose a profile or enter a name on the specific review window you want to assign. Nothing is selected implicitly, and other windows remain unchanged.";
    const hiddenNote = [
      hiddenShortWindows ? `${hiddenShortWindows} isolated sub-second fragment${hiddenShortWindows === 1 ? " is" : "s are"} hidden because a longer window is available` : "",
      omittedWindows ? `${omittedWindows} additional review window${omittedWindows === 1 ? " is" : "s are"} omitted; Echo shows at most 10 examples here` : "",
    ].filter(Boolean).join(". ");
    const segmentList = reviewWindows.map((window) => {
      const first = window.segments[0];
      const assigned = window.segments.find((segment) => segment.voice_id);
      const windowState = assigned?.mapping_state || first.mapping_state || "anonymous";
      const ids = window.segments.map((segment) => segment.segment_id).join(",");
      const transcript = window.transcript || "No matching transcript snippet was found for this timestamp.";
      return `<article class="work-detail echo-segment ${window.id === selectedWindow?.id ? "echo-segment-selected" : ""}"><div class="work-detail-head"><strong>${escapeHTML(formatEchoSeconds(window.start_seconds))}–${escapeHTML(formatEchoSeconds(window.end_seconds))}</strong>${stateChip(windowState)}</div><p>Anonymous label: <code>${escapeHTML(first.anonymous_speaker)}</code>${assigned ? ` · voice profile: <code>${escapeHTML(assigned.voice_id)}</code>` : ""}</p><p><b>Transcript:</b> ${escapeHTML(transcript)}</p><div class="button-row"><button type="button" class="button quiet" data-echo-play-segment="${escapeHTML(window.id)}" data-echo-play-start="${window.start_seconds}" data-echo-play-end="${window.end_seconds}">Play this window</button><button type="button" class="button quiet" data-echo-select-segment="${escapeHTML(window.id)}" data-echo-speaker-for-segment="${escapeHTML(speaker)}">Use this window</button>${assigned ? `<button type="button" class="button quiet" data-echo-clear-segment="${escapeHTML(assigned.segment_id)}">Clear this assignment</button>` : ""}</div><form class="echo-segment-identity" data-echo-segment-form="${escapeHTML(window.id)}" data-echo-segment-ids="${escapeHTML(ids)}" data-echo-segment-speaker="${escapeHTML(speaker)}"><label>Assign this review window</label><div class="echo-segment-controls"><select data-echo-profile-for="${escapeHTML(window.id)}"><option value="">Existing profile (optional)</option>${(body.profiles || []).map((profile) => `<option value="${escapeHTML(profile.voice_id)}">${escapeHTML(profile.name)}</option>`).join("")}</select><input type="text" data-echo-name-for="${escapeHTML(window.id)}" placeholder="Human name" autocomplete="off" required><button type="submit" class="button">Save this window</button></div><p class="form-message" data-echo-status-for="${escapeHTML(window.id)}" role="status" aria-live="polite"></p></form><div class="zodiac-meta">${window.segments.length} underlying diarized interval${window.segments.length === 1 ? "" : "s"} · source ${escapeHTML(source)}</div></article>`;
    }).join("") || `<p class="muted">No review windows for this speaker are currently in the visible Echo snapshot. Longer speech windows are prioritized over isolated fragments.</p>`;
    const approval = confirmedSample ? `<div class="button-row"><button type="button" class="button" data-echo-approve-speaker="${escapeHTML(speaker)}" data-echo-approve-segment="${escapeHTML(confirmedSample.segment_id)}">Approve this speaker label for this run</button><span class="muted">This will apply ${escapeHTML((body.segments || []).filter((segment) => segment.anonymous_speaker === speaker && !segment.voice_id).length)} remaining anonymous snippets to ${escapeHTML((body.profiles || []).find((profile) => profile.voice_id === confirmedSample.voice_id)?.name || confirmedSample.voice_id)} in ${escapeHTML(run.run_id || "this run")}; it requires explicit confirmation.</span></div>` : `<p class="muted">Save one snippet first if you later decide this anonymous label is consistent enough to approve in bulk.</p>`;
    $("echo-speaker-detail-body").innerHTML = `<article class="work-detail"><div class="work-detail-head"><strong>Review substantial speech windows</strong>${stateChip("human confirmation")}</div><p>Echo groups nearby speech from the same anonymous speaker into review windows up to 10 seconds and shows at most 10 examples. Listen before assigning it. Saving enrolls one larger local sample and applies the assignment to the underlying diarized intervals in that window; short fragments are supporting evidence, not separate focal samples.</p>${hiddenNote ? `<p class="muted echo-review-note">${escapeHTML(hiddenNote)}.</p>` : ""}${approval}</article>${segmentList}`;
    document.querySelectorAll("#echo-speaker-detail [data-echo-select-segment], #echo-speaker-detail .echo-segment-selected").forEach((element) => {
      if (element.matches("[data-echo-select-segment]")) element.remove();
      else element.classList.remove("echo-segment-selected");
    });
    const bulkButton = document.querySelector("#echo-speaker-detail [data-echo-approve-speaker]");
    if (bulkButton) bulkButton.textContent = `Assign remaining ${speaker} snippets`;
    const bulkNote = bulkButton?.parentElement?.querySelector(".muted");
    if (bulkNote) bulkNote.textContent = `Applies only to ${speaker} in ${run.run_id || "this run"}. Other speakers and runs are untouched.`;
    if (includeAssigned && selectedSegmentID) {
      const selected = segments.find((segment) => segment.segment_id === selectedSegmentID);
      const form = document.querySelector(`[data-echo-segment-form="${CSS.escape(selectedSegmentID)}"]`);
      const profile = (body.profiles || []).find((item) => item.voice_id === selected?.voice_id);
      if (form && selected?.voice_id) {
        form.querySelector("[data-echo-profile-for]").value = selected.voice_id;
        form.querySelector("[data-echo-name-for]").value = profile?.name || "";
      }
    }
    $("echo-speaker-detail").scrollIntoView({ behavior: "smooth", block: "nearest" });
  }
  function playEchoSegment(segmentID, startOverride = "", endOverride = "") {
    const body = state.echo || {};
    const run = body.latest_run || {};
    const segment = (body.segments || []).find((item) => item.segment_id === segmentID);
    const player = $("echo-snippet-player");
    if (!run.run_id || !segment || !player) return;
    player.hidden = false;
    const start = startOverride === "" ? segment.start_seconds : Number(startOverride);
    const end = endOverride === "" ? segment.end_seconds : Number(endOverride);
    player.src = "/v1/echo/runs/" + encodeURIComponent(run.run_id) + "/audio#t=" + Number(start).toFixed(3) + "," + Number(end).toFixed(3);
    player.load();
    player.play().catch(() => showNotice("Use the audio controls to start this snippet.", true));
  }
  async function saveEchoSegmentIdentity(form) {
    const body = state.echo || {};
    const run = body.latest_run || {};
    const segmentID = form.dataset.echoSegmentForm;
    const segmentIDs = (form.dataset.echoSegmentIds || segmentID).split(",").filter(Boolean);
    const speaker = form.dataset.echoSegmentSpeaker;
    const windowSegments = (body.segments || []).filter((segment) => segmentIDs.includes(segment.segment_id));
    const sample = windowSegments.slice().sort((a, b) => echoSegmentDuration(b) - echoSegmentDuration(a))[0];
    const profileID = form.querySelector("[data-echo-profile-for]")?.value || "";
    const selectedProfile = (body.profiles || []).find((profile) => profile.voice_id === profileID);
    const nameInput = form.querySelector("[data-echo-name-for]");
    const name = nameInput?.value.trim() || selectedProfile?.name || "";
    const status = form.querySelector("[data-echo-status-for]");
    const submit = form.querySelector("button[type=submit]");
    if (!speaker || !run.run_id || !run.source_path || !sample || !name) { if (status) { status.textContent = "Choose a profile or enter a human name for this review window."; status.classList.add("error"); } return; }
    if (submit) { submit.disabled = true; submit.textContent = "Saving…"; }
    try {
      const fame = await json("/v1/fame");
      let person = (fame.people || []).find((item) => String(item.preferred_name || "").toLowerCase() === name.toLowerCase());
      if (!person) person = await post("/v1/fame/people", { preferred_name: name, status: "confirmed" });
      let profile = selectedProfile || (body.profiles || []).find((item) => String(item.name || "").toLowerCase() === name.toLowerCase());
      if (!profile) profile = await post("/v1/echo/voices", { name });
      const windowStart = Math.min(...windowSegments.map((segment) => Number(segment.start_seconds)));
      const windowEnd = Math.max(...windowSegments.map((segment) => Number(segment.end_seconds)));
      await post(`/v1/echo/voices/${encodeURIComponent(profile.voice_id)}/samples`, { source_path: run.source_path, start_seconds: windowStart, end_seconds: windowEnd });
      for (const segment of windowSegments) await post(`/v1/echo/segments/${encodeURIComponent(segment.segment_id)}/confirm`, { voice_id: profile.voice_id });
      const fameAfterPerson = await json("/v1/fame");
      if (!(fameAfterPerson.conversations || []).some((conversation) => conversation.run_id === run.run_id)) await post("/v1/fame/conversations", { run_id: run.run_id });
      for (const segment of windowSegments) {
        await post("/v1/fame/corrections", { target_type: "segment", target_id: segment.segment_id, kind: "operator_identity_confirmation", statement: `Operator identified this review window of ${speaker} as ${name} in Echo run ${run.run_id}.`, person_id: person.person_id });
        await post(`/v1/fame/people/${encodeURIComponent(person.person_id)}/evidence`, { kind: "operator_correction", source_ref: `${run.run_id}:${segment.segment_id}`, detail: `Human entered ${name} for a larger Echo review window and confirmed the local voice sample.`, confidence: 1 });
      }
      showNotice(`This ${windowEnd - windowStart < 1 ? "fragment" : "review window"} is now assigned to ${name}. Short intervals were not enrolled separately.`);
      await loadEcho(run.run_id);
      $("echo-speaker-detail").hidden = true;
    } catch (error) {
      if (status) { status.textContent = error.message || "This snippet could not be saved."; status.classList.add("error"); }
      showNotice(error.message || "This snippet could not be saved. No cloud upload was attempted.", true);
    } finally {
      if (submit) { submit.disabled = false; submit.textContent = "Save this snippet"; }
    }
  }
  async function setEchoSpeakerIdentity(event) {
    event.preventDefault();
    const form = event.currentTarget;
    const speaker = form.dataset.echoSpeaker;
    const name = $("echo-speaker-identity-name").value.trim();
    const selectedVoiceID = $("echo-speaker-identity-profile").value;
    const body = state.echo || {};
    const run = body.latest_run || {};
    const segmentID = form.dataset.echoSegment;
    const segments = (body.segments || []).filter((segment) => segment.anonymous_speaker === speaker);
    const sample = segments.find((segment) => segment.segment_id === segmentID);
    if (!speaker || !name || !run.run_id || !run.source_path || !sample) { showNotice("Select one transcript snippet before setting its identity. No bulk assignment is available.", true); return; }
    const submit = form.querySelector("button[type=submit]");
    submit.disabled = true;
    submit.textContent = "Saving…";
    try {
      const fame = await json("/v1/fame");
      let person = (fame.people || []).find((item) => String(item.preferred_name || "").toLowerCase() === name.toLowerCase());
      if (!person) person = await post("/v1/fame/people", { preferred_name: name, status: "confirmed" });
      let profile = (body.profiles || []).find((item) => item.voice_id === selectedVoiceID) || (body.profiles || []).find((item) => String(item.name || "").toLowerCase() === name.toLowerCase());
      if (!profile) profile = await post("/v1/echo/voices", { name });
      await post(`/v1/echo/voices/${encodeURIComponent(profile.voice_id)}/samples`, { source_path: run.source_path, start_seconds: sample.start_seconds, end_seconds: sample.end_seconds });
      await post(`/v1/echo/segments/${encodeURIComponent(sample.segment_id)}/confirm`, { voice_id: profile.voice_id });
      const fameAfterPerson = await json("/v1/fame");
      if (!(fameAfterPerson.conversations || []).some((conversation) => conversation.run_id === run.run_id)) await post("/v1/fame/conversations", { run_id: run.run_id });
      await post("/v1/fame/corrections", { target_type: "segment", target_id: sample.segment_id, kind: "operator_identity_confirmation", statement: `Operator identified this segment of ${speaker} as ${name} in Echo run ${run.run_id}.`, person_id: person.person_id });
      await post(`/v1/fame/people/${encodeURIComponent(person.person_id)}/evidence`, { kind: "operator_correction", source_ref: `${run.run_id}:${sample.segment_id}`, detail: `Human entered ${name} for one Echo segment and confirmed the local voice sample.`, confidence: 1 });
      $("echo-identity-status").textContent = `Saved ${speaker} as ${name}.`;
      $("echo-identity-status").classList.remove("error");
      showNotice(`One ${speaker} segment is now assigned to ${name}. Other segments remain unchanged.`);
      await loadEcho(run.run_id);
      renderEchoSpeakerDetail(speaker, sample.segment_id);
    } catch (error) {
      $("echo-identity-status").textContent = error.message || "The identity could not be saved.";
      $("echo-identity-status").classList.add("error");
      showNotice(error.message || "The speaker identity could not be saved. No cloud upload was attempted.", true);
    } finally {
      submit.disabled = false;
      submit.textContent = "Set speaker identity";
    }
  }
  async function clearEchoSegment(segmentID) {
    if (!segmentID) return;
    try {
      await post(`/v1/echo/segments/${encodeURIComponent(segmentID)}/clear`);
      const fame = await json("/v1/fame");
      if ((fame.attributions || []).some((item) => item.segment_id === segmentID && item.person_id)) await post(`/v1/fame/segments/${encodeURIComponent(segmentID)}/clear`);
      showNotice("That segment is anonymous again. Other segment assignments were left unchanged.");
      const speaker = document.querySelector(`[data-echo-segment-form="${CSS.escape(segmentID)}"]`)?.dataset.echoSegmentSpeaker || "";
      await loadEcho(state.echo?.selected_run_id || "");
      if (speaker) renderEchoSpeakerDetail(speaker, segmentID);
    } catch (error) {
      showNotice(error.message || "The segment assignment could not be cleared.", true);
    }
  }
  async function renameEchoVoice(voiceID) {
    const input = document.querySelector(`[data-echo-voice-name="${CSS.escape(voiceID)}"]`);
    const name = input?.value.trim();
    if (!name) { showNotice("Enter a name before saving the voice profile.", true); return; }
    try {
      await post(`/v1/echo/voices/${encodeURIComponent(voiceID)}/rename`, { name });
      showNotice(`Voice profile renamed to ${name}. Existing segment assignments remain attached to this profile.`);
      await loadEcho(state.echo?.selected_run_id || "");
    } catch (error) {
      showNotice(error.message || "The voice profile name could not be saved.", true);
    }
  }
  async function archiveEchoVoice(voiceID) {
    if (!window.confirm("Remove this voice profile from the active Echo list? Historical samples and assignments will be preserved.")) return;
    try {
      await post(`/v1/echo/voices/${encodeURIComponent(voiceID)}/archive`);
      showNotice("Voice profile removed from the active list. Historical evidence was preserved.");
      await loadEcho(state.echo?.selected_run_id || "");
    } catch (error) {
      showNotice(error.message || "The voice profile could not be removed.", true);
    }
  }
  async function approveEchoSpeaker(speaker, segmentID) {
    const body = state.echo || {};
    const run = body.latest_run || {};
    const segment = (body.segments || []).find((item) => item.anonymous_speaker === speaker && item.segment_id === segmentID);
    const profile = (body.profiles || []).find((item) => item.voice_id === segment?.voice_id);
    const name = profile?.name || segment?.voice_id || "";
    if (!speaker || !run.run_id || !name || !segment?.voice_id) { showNotice("Select and save one identified snippet before approving this speaker label.", true); return; }
    if (!window.confirm(`Approve ${speaker} as ${name} for every segment in ${run.run_id}? Review the snippets first; this changes the whole anonymous label only in this run.`)) return;
    try {
      const fame = await json("/v1/fame");
      let person = (fame.people || []).find((item) => String(item.preferred_name || "").toLowerCase() === name.toLowerCase());
      if (!person) person = await post("/v1/fame/people", { preferred_name: name, status: "confirmed" });
      const bulkResult = await post(`/v1/echo/voices/${encodeURIComponent(segment.voice_id)}/confirm`, { run_id: run.run_id, speaker });
      const fameAfterPerson = await json("/v1/fame");
      if (!(fameAfterPerson.conversations || []).some((conversation) => conversation.run_id === run.run_id)) await post("/v1/fame/conversations", { run_id: run.run_id });
      await post("/v1/fame/corrections", { target_type: "speaker", target_id: run.run_id, kind: "operator_speaker_label_confirmation", statement: `Operator approved every ${speaker} segment in Echo run ${run.run_id} as ${name}.`, person_id: person.person_id, run_id: run.run_id, speaker });
      showNotice(`${bulkResult.confirmed || 0} ${speaker} segment${bulkResult.confirmed === 1 ? "" : "s"} assigned to ${name} in ${run.run_id}. Other speakers and runs remain unchanged.`);
      await loadEcho(run.run_id);
      renderEchoSpeakerDetail(speaker, segment.segment_id);
    } catch (error) {
      showNotice(error.message || "The speaker label could not be approved.", true);
    }
  }
  async function loadEcho(runID = "") { try { const url = runID ? `/v1/gui/echo?run_id=${encodeURIComponent(runID)}` : "/v1/gui/echo"; renderEcho(await json(url)); } catch (error) { $("echo-run").innerHTML = `<p class="muted">Echo state is unavailable; no voice action was taken.</p>`; showNotice(error.message || "Echo state unavailable.", true); } }
  function renderFame(body) {
    state.fame = body;
    const people = Array.isArray(body.people) ? body.people : [];
    const conversations = Array.isArray(body.conversations) ? body.conversations : [];
    const attributions = Array.isArray(body.attributions) ? body.attributions : [];
    const observations = Array.isArray(body.observations) ? body.observations : [];
    const corrections = Array.isArray(body.corrections) ? body.corrections : [];
    $("fame-state").outerHTML = `<span id="fame-state" class="state-chip enabled">local substrate</span>`;
    renderMetrics("fame-summary", { "people": people.length, "conversations": conversations.length, "anonymous segments": attributions.filter((item) => !item.person_id).length, "accountable observations": observations.length, "human corrections": corrections.length });
    $("fame-boundary-note").textContent = (body.boundaries || ["Identity evidence is inspectable."])[0];
    $("fame-people-count").textContent = `${people.length} record${people.length === 1 ? "" : "s"}`;
    $("fame-people").innerHTML = people.map((person) => `<article class="work-detail"><div class="work-detail-head"><strong>${escapeHTML(person.preferred_name)}</strong>${stateChip(person.status)}</div><p>${person.is_self ? "Local owner identity." : "Durable person reference; identity may remain provisional."}</p><div class="zodiac-meta">${escapeHTML(person.person_id)}${person.aliases?.length ? ` · aliases ${escapeHTML(person.aliases.join(", "))}` : ""}</div></article>`).join("") || `<p class="muted">No people registered.</p>`;
    $("fame-conversation-count").textContent = `${conversations.length} record${conversations.length === 1 ? "" : "s"}`;
    $("fame-conversations").innerHTML = conversations.map((conversation) => { const linked = attributions.filter((item) => item.conversation_id === conversation.conversation_id); const named = linked.filter((item) => item.person_id).length; return `<article class="work-detail"><div class="work-detail-head"><strong>${escapeHTML(conversation.conversation_id)}</strong>${stateChip(conversation.boundary_state)}</div><p>${escapeHTML(conversation.source_path)}</p><div class="zodiac-meta">${escapeHTML(conversation.run_id || "no Echo run")} · ${linked.length} segment links · ${named} resolved · status ${escapeHTML(conversation.status)}</div></article>`; }).join("") || `<p class="muted">No Fame conversation boundaries recorded.</p>`;
    $("fame-observation-count").textContent = `${observations.length} record${observations.length === 1 ? "" : "s"}`;
    $("fame-observations").innerHTML = observations.map((item) => `<article class="work-detail"><div class="work-detail-head"><strong>${escapeHTML(item.kind)}</strong>${stateChip(item.status)}</div><p>${escapeHTML(item.content)}</p><div class="zodiac-meta">source ${escapeHTML(item.source_ref)} · confidence ${escapeHTML(item.confidence)}</div></article>`).join("") || `<p class="muted">No accountable interaction observations recorded.</p>`;
    $("fame-correction-count").textContent = `${corrections.length} record${corrections.length === 1 ? "" : "s"}`;
    $("fame-corrections").innerHTML = corrections.map((item) => `<article class="work-detail"><div class="work-detail-head"><strong>${escapeHTML(item.kind)}</strong>${stateChip("human correction")}</div><p>${escapeHTML(item.statement)}</p><div class="zodiac-meta">${escapeHTML(item.target_type)}${item.target_id ? ` · ${escapeHTML(item.target_id)}` : ""} · ${escapeHTML(item.created_at)}</div></article>`).join("") || `<p class="muted">No human corrections recorded.</p>`;
  }
  async function loadFame() { try { renderFame(await json("/v1/gui/fame")); } catch (error) { $("fame-people").innerHTML = `<p class="muted">Fame state is unavailable; no identity action was taken.</p>`; showNotice(error.message || "Fame state unavailable.", true); } }
  const workCategoryLabels = { all: "All identified work", active: "Active work", waiting: "Waiting work", needs_grant: "Needs Grant", blocked: "Blocked work", needs_attention: "Needs decision", upcoming: "Upcoming known work" };
  const renderWorkItems = (items, includeViewButton = true) => items.map((item) => `<article class="work-detail"><div class="work-detail-head"><div><strong>${escapeHTML(item.title)}</strong><div class="work-detail-meta">${stateChip(item.category)} ${stateChip(item.state)}</div></div><span class="work-detail-id">${escapeHTML(item.id)}</span></div><p><b>Why it is here:</b> ${escapeHTML(item.explanation)}</p><p><b>Next safe step:</b> ${escapeHTML(item.next_action)}</p>${includeViewButton ? `<button type="button" class="button quiet" data-view-link="${escapeHTML(item.view)}">Open ${escapeHTML(item.view)} view →</button>` : (item.category === "blocked" ? `<span class="no-safe-action">No safe execution action is available until this boundary is resolved.</span>` : `<button type="button" class="button quiet" data-view-link="${escapeHTML(item.view)}">Review in ${escapeHTML(item.view)} →</button>`)}</article>`).join("");
  async function loadWorkContext(category) { const context = $("work-context"); context.hidden = false; $("work-context-title").textContent = workCategoryLabels[category] || "Work context"; $("work-context-intro").textContent = "Loading the underlying records and the next safe step…"; $("work-context-items").innerHTML = ""; context.scrollIntoView({ behavior: "smooth", block: "nearest" }); try { const body = await json(`/v1/gui/work?category=${encodeURIComponent(category)}`); const items = Array.isArray(body.items) ? body.items : []; $("work-context-intro").textContent = items.length ? `${items.length} bounded record${items.length === 1 ? "" : "s"} identified. Each entry says why it is here and what to do next.` : "No underlying record is currently identified for this category. The count may have changed; refresh the live state."; $("work-context-items").innerHTML = renderWorkItems(items) || `<div class="empty-state compact"><h2>Nothing currently needs this view</h2><p>Refresh the live state if you expected a record here.</p></div>`; } catch (error) { console.warn("[Panopticon] Work context unavailable", error); $("work-context-intro").textContent = "Work context is temporarily unavailable. The rest of Panopticon remains available."; $("work-context-items").innerHTML = `<div class="empty-state compact"><h2>Work details unavailable</h2><p>Retry the live state shortly; no action was taken.</p><button type="button" class="button quiet" data-work-category="${escapeHTML(category)}">Retry work details</button></div>`; } }
  async function loadExecutionWork(category) { const target = $("execution-work-list"); if (!target) return; target.innerHTML = `<p class="muted">Loading ${escapeHTML(workCategoryLabels[category] || "work")}…</p>`; try { const body = await json(`/v1/gui/work?category=${encodeURIComponent(category)}`); const items = Array.isArray(body.items) ? body.items : []; target.innerHTML = renderWorkItems(items, false) || `<div class="empty-state compact"><h2>No records in this view</h2><p>Choose another filter or return to Home to review the work categories.</p></div>`; } catch (error) { console.warn("[Panopticon] Execution work unavailable", error); target.innerHTML = `<div class="empty-state compact"><h2>Execution records unavailable</h2><p>The live work list could not be read. No execution action was taken.</p><button type="button" class="button quiet" id="retry-execution">Retry execution records</button></div>`; $("retry-execution")?.addEventListener("click", () => loadExecutionWork(category)); } }
  const mailStateLabel = { unclassified: "new / not yet understood", awaiting_apollo: "waiting for Apollo", understood: "understood", deterministically_rejected: "deterministically filtered", blocked: "blocked by a safety boundary", failed: "failed; inspect before retrying" };
  const mailMessageHTML = (message) => { const stateText = mailStateLabel[message.state] || message.state || "unknown"; const summary = message.summary || (message.state === "unclassified" ? "New mail has been accounted for and is awaiting the next explicit mail check." : "No summary is available yet."); const details = [["Sender", message.sender || "unknown sender"], ["Classification", message.classification || "not classified"], ["Priority", message.priority || "not assigned"], ["Reply-worthiness", message.reply_worthiness || "not assessed"], ["Requested action", message.requested_action || "none recorded"], ["Deadline / time", message.deadline || "none recorded"], ["Mailbox suggestion", message.suggested_mailbox_action || "none; understanding is not authorization"], ["Evidence", `${(message.source_citations || []).length} citation${(message.source_citations || []).length === 1 ? "" : "s"}`]]; if (message.suspicion_state) details.push(["Suspicion", message.suspicion_state]); if (message.attachment_state) details.push(["Attachments", message.attachment_state]); return `<article class="mail-message"><div class="mail-message-head"><div><strong>${escapeHTML(message.subject || "(no subject)")}</strong><p>${escapeHTML(message.sender || "unknown sender")} · ${escapeHTML(message.received_at || "date unavailable")}</p></div>${stateChip(stateText)}</div><p class="mail-summary">${escapeHTML(summary)}</p><div class="mail-detail-grid">${details.map(([label, value]) => `<div><span>${escapeHTML(label)}</span><strong>${escapeHTML(value)}</strong></div>`).join("")}</div>${message.commitment ? `<p><b>Commitment:</b> ${escapeHTML(message.commitment)}</p>` : ""}${message.last_error ? `<p class="mail-warning"><b>Boundary:</b> ${escapeHTML(message.last_error)}</p>` : ""}<p class="mail-provenance">Source identity: <code>${escapeHTML(message.source_id)}</code>${message.apollo_report_id ? ` · Apollo report <code>${escapeHTML(message.apollo_report_id)}</code>` : ""}</p></article>`; };
  function renderMailHome() { const target = $("mail-home-summary"); if (!target) return; const mail = state.mail; if (!mail) { target.innerHTML = `<div class="empty-state compact"><h2>Mail state unavailable</h2><p>Other Panopticon state remains available. Open Mail and retry when the local mail surface is ready.</p></div>`; $("mail-home-note").textContent = "Mail state is unknown; no healthy mailbox state is implied."; return; } const counts = mail.state_counts || {}; const waiting = (Number(counts.unclassified) || 0) + (Number(counts.awaiting_apollo) || 0); const blocked = (Number(counts.blocked) || 0) + (Number(counts.failed) || 0); renderMetrics("mail-home-summary", { "new or waiting": waiting, "understood": counts.understood || 0, "blocked / safety": blocked, "filtered with provenance": counts.deterministically_rejected || 0 }); $("mail-home-note").textContent = mail.enabled ? `${mail.account || "Configured account"} · ${mail.inbox || "configured inbox"} · Check mail now is explicit and bounded. No send or delete operation is available.` : "Mail processing is disabled by configuration; no healthy mailbox state is implied."; }
  function renderMail(body) { state.mail = body; const counts = body.state_counts || {}; const enabled = Boolean(body.enabled); $("mail-state").outerHTML = `<span id="mail-state" class="state-chip ${enabled ? "enabled" : "disabled"}">${enabled ? "ready for explicit check" : "disabled"}</span>`; renderMetrics("mail-summary", { "new / unclassified": counts.unclassified || 0, "waiting for Apollo": counts.awaiting_apollo || 0, "understood": counts.understood || 0, "filtered": counts.deterministically_rejected || 0, "blocked / failed": (Number(counts.blocked) || 0) + (Number(counts.failed) || 0), "action receipts": (body.actions || []).length }); const last = body.last_run || {}; $("mail-note").textContent = last.run_id ? `Last explicit round ${last.run_id}: ${last.new_processed || 0} new and ${last.backlog_processed || 0} backlog messages processed. Cursor: ${body.backlog?.cursor || "not established"}.` : "No explicit mail round has completed. Check mail now will account for new mail before bounded backlog work."; const messages = Array.isArray(body.messages) ? body.messages : []; $("mail-message-count").textContent = `${messages.length} recorded`; $("mail-messages").innerHTML = messages.map(mailMessageHTML).join("") || `<div class="empty-state compact"><h2>No mail recorded</h2><p>Check mail now to inspect the configured account. Credentials remain outside this surface.</p></div>`; const rules = Array.isArray(body.rules) ? body.rules : []; $("mail-rule-count").textContent = `${rules.length} registered`; $("mail-rules").innerHTML = rules.map((rule) => `<article class="mail-rule"><div><strong>${escapeHTML(rule.rule_type)}: ${escapeHTML(rule.match_value)}</strong>${stateChip(rule.state)}</div><p>${escapeHTML(rule.behavior)} · matched ${escapeHTML(rule.match_count)} time${rule.match_count === 1 ? "" : "s"}</p>${rule.state === "enabled" ? `<button class="button quiet" data-mail-rule-disable="${escapeHTML(rule.rule_id)}">Disable rule</button>` : ""}</article>`).join("") || `<p class="muted">No deterministic rejection rules are registered.</p>`; const actions = Array.isArray(body.actions) ? body.actions : []; $("mail-actions").innerHTML = actions.map((action) => `<article class="mail-action"><div><strong>${escapeHTML(action.operation)}</strong>${stateChip(action.outcome)}</div><p>${escapeHTML(action.source_id)} · ${escapeHTML(action.result || action.error || "No receipt detail")}</p></article>`).join("") || `<p class="muted">No mailbox action receipts. Suggestions remain non-authorizing until Future and Mercury lawfully resolve them.</p>`; document.querySelectorAll("[data-mail-rule-disable]").forEach((button) => button.addEventListener("click", () => disableMailRule(button.dataset.mailRuleDisable))); renderMailHome(); }
  async function loadMail() { try { renderMail(await json("/v1/gui/mail")); } catch (error) { console.warn("[Panopticon] Mail surface unavailable", error); $("mail-state").outerHTML = `<span id="mail-state" class="state-chip unknown">unknown</span>`; $("mail-summary").innerHTML = `<div class="empty-state compact"><h2>Mail information unavailable</h2><p>The local mail surface could not be read. No mail action was taken.</p><button id="retry-mail" class="button quiet">Retry mail information</button></div>`; $("mail-messages").innerHTML = ""; $("retry-mail")?.addEventListener("click", loadMail); } }
  const zodiacItemHTML = (item) => `<article class="zodiac-item"><div class="zodiac-item-head"><strong>${escapeHTML(item.schedule_id || item.name || "Zodiac item")}</strong>${stateChip(item.state)}</div><p>${escapeHTML(item.last_error || `Due ${item.due_at || "at the declared time"}.`)} </p><div class="zodiac-meta">${escapeHTML(item.occurrence_key || item.instance_id || "persisted instance")}</div></article>`;
  function renderZodiac(body, today, needs) { const schedules = Array.isArray(body.schedules) ? body.schedules : []; const recurring = Array.isArray(body.recurring) ? body.recurring : []; const todayItems = Array.isArray(today?.items) ? today.items : []; const needsItems = Array.isArray(needs?.items) ? needs.items : []; $("zodiac-state").outerHTML = `<span id="zodiac-state" class="state-chip ${body.enabled ? "enabled" : "disabled"}">${body.enabled ? (body.paused ? "paused" : "running") : "disabled"}</span>`; $("zodiac-today").innerHTML = todayItems.map(zodiacItemHTML).join("") || `<div class="empty-state compact"><h2>No due items recorded today</h2><p>Evaluate now when you want Zodiac to inspect the persisted declarations.</p></div>`; $("zodiac-needs-count").textContent = `${needsItems.length} exception${needsItems.length === 1 ? "" : "s"}`; $("zodiac-needs").innerHTML = needsItems.map(zodiacItemHTML).join("") || `<div class="empty-state compact"><h2>Nothing needs your decision</h2><p>Ambiguity, conflicts, authority gaps, and unavailable adapters are surfaced here rather than guessed through.</p></div>`; $("zodiac-recurring").innerHTML = recurring.map((item) => `<article class="zodiac-item"><div class="zodiac-item-head"><strong>${escapeHTML(item.name)}</strong>${stateChip(item.status)}</div><p>${escapeHTML(item.recurrence)} · next due ${escapeHTML(item.next_due_at || "not yet evaluated")} · catch-up ${escapeHTML(item.catch_up_policy)}</p><div class="zodiac-meta">${escapeHTML(item.source)} · ${escapeHTML(item.action)}</div></article>`).join("") || `<p class="muted">No recurring declarations are registered.</p>`; $("zodiac-providers").innerHTML = (body.providers || []).map((provider) => `<article class="zodiac-provider"><div class="zodiac-provider-head"><strong>${escapeHTML(provider.provider)} calendar</strong>${stateChip(provider.state)}</div><p>${escapeHTML(provider.reason || "No provider detail")}</p><div class="zodiac-meta">${escapeHTML(provider.capability_id)} · credentials ${escapeHTML(provider.credentials || "omitted")}</div></article>`).join(""); }
  async function loadZodiac() { try { const [body, today, needs] = await Promise.all([json("/v1/gui/zodiac"), json("/v1/gui/zodiac/today"), json("/v1/gui/zodiac/needs-you")]); renderZodiac(body, today, needs); } catch (error) { console.warn("[Panopticon] Zodiac surface unavailable", error); $("zodiac-state").outerHTML = `<span id="zodiac-state" class="state-chip unknown">unknown</span>`; $("zodiac-today").innerHTML = `<div class="empty-state compact"><h2>Zodiac information unavailable</h2><p>The persisted schedule surface could not be read. No schedule action was taken.</p><button id="retry-zodiac" class="button quiet">Retry Zodiac information</button></div>`; $("retry-zodiac")?.addEventListener("click", loadZodiac); } }
  async function evaluateZodiac() { const button = $("zodiac-round"); button.disabled = true; button.textContent = "Evaluating…"; try { const result = await post("/v1/gui/zodiac/round"); showNotice(result.result || "Zodiac evaluated persisted schedules."); await loadZodiac(); await refreshState(); } catch (error) { showNotice(error.message || "Zodiac evaluation could not complete.", true); } finally { button.disabled = false; button.textContent = "Evaluate now"; } }
  async function checkMail() { const button = $("mail-round"); button.disabled = true; button.textContent = "Checking…"; try { const result = await post("/v1/gui/mail/round"); showNotice(`Mail check completed: ${result.new_processed || 0} new and ${result.backlog_processed || 0} backlog messages processed.`); await loadMail(); await refreshState(); } catch (error) { showNotice(error.message || "Mail check could not complete. No mailbox action was implied.", true); } finally { button.disabled = false; button.textContent = "Check mail now"; } }
  async function disableMailRule(id) { if (!window.confirm("Disable this deterministic rejection rule? Future mail will remain accounted for and may be considered by Apollo.")) return; try { await post(`/v1/gui/mail/rules/${encodeURIComponent(id)}/disable`); await loadMail(); } catch (error) { showNotice(error.message || "Rule could not be disabled.", true); } }
  async function loadNotifications() { try { const body = await json("/v1/gui/notifications"); $("notification-source").textContent = body.source || "local notification adapter"; $("notifications-list").innerHTML = (body.notifications || []).map((n) => `<article class="notification"><div class="notification-head"><strong>${escapeHTML(n.message || "Notification")}</strong><time>${escapeHTML(n.occurred_at)}</time></div><p>${stateChip(n.outcome)} ${escapeHTML(n.result || n.error || "")}</p></article>`).join("") || `<div class="empty-state"><h2>No notifications</h2><p>The local notification audit has no entries yet.</p></div>`; } catch (error) { showNotice(error.message, true); } }
  async function loadRumor() { try { state.rumor = await json("/v1/rumor"); const r = state.rumor; $("rumor-state").outerHTML = `<span id="rumor-state" class="state-chip ${r.enabled ? "enabled" : "disabled"}">${r.enabled ? "enabled" : "disabled"}</span>`; renderMetrics("rumor-summary", { "watch paths": (r.watch_paths || []).length, "queued": r.counts?.queued || 0, "live transcription": "off" }); const sources = r.recent_sources || []; $("rumor-source-count").textContent = `${sources.length} recent`; $("rumor-sources").innerHTML = sources.map((s) => `<article class="source"><div class="source-head"><strong>${escapeHTML(s.source_filename || s.source_id || "source")}</strong>${stateChip(s.state)}</div><p>${escapeHTML(s.source_path || s.last_error || "No additional source detail")}</p><div class="button-row source-actions"><button class="button quiet" data-rumor-action="retry" data-source-id="${escapeHTML(s.source_id)}">Retry</button><button class="button quiet" data-rumor-action="ignore" data-source-id="${escapeHTML(s.source_id)}">Ignore</button></div></article>`).join("") || `<div class="empty-state"><h2>No recent sources</h2><p>Rumor is watching local configured paths only.</p></div>`; document.querySelectorAll("[data-rumor-action]").forEach((b) => b.addEventListener("click", () => rumorSourceAction(b.dataset.rumorAction, b.dataset.sourceId))); } catch (error) { showNotice(error.message, true); } }
  const renderCapabilitiesUnavailable = (detail) => { console.warn("[Panopticon] Capability information unavailable", detail); $("capability-state").outerHTML = `<span id="capability-state" class="state-chip unknown">unknown</span>`; $("capabilities").innerHTML = `<div class="empty-state capability-unavailable"><span>?</span><h2>Capability information unavailable</h2><p>Bifrost could not provide capability state. Other Panopticon views remain available.</p><button id="retry-capabilities" class="button quiet">Retry capability information</button></div>`; $("retry-capabilities").addEventListener("click", loadCapabilities); };
  async function loadCapabilities() { try { const body = await json("/v1/bifrost/capabilities"); const capabilities = Array.isArray(body) ? body : (body && Array.isArray(body.capabilities) ? body.capabilities : null); if (!capabilities) { renderCapabilitiesUnavailable("response was not a capability list"); return; } $("capability-state").outerHTML = `<span id="capability-state" class="state-chip ready">available</span>`; $("capabilities").innerHTML = capabilities.map((c) => { const id = c.capability_id || c.id; const approval = c.use_approved ? "use approved" : (c.registration_approved ? "registered approval" : "approval required"); return `<article class="capability"><div class="capability-head"><h3>${escapeHTML(c.name || id)}</h3>${stateChip(c.enabled ? "enabled" : "disabled")}</div><p>${escapeHTML(c.description || c.scope || "Registered capability")}</p><div class="capability-meta"><span>registered: yes</span><span>approval: ${escapeHTML(approval)}</span><span>download/install: unavailable</span></div><div class="button-row capability-actions"><button class="button quiet" data-cap-action="disable" data-cap-id="${escapeHTML(id)}" ${!c.enabled ? "disabled" : ""}>Disable</button><button class="button" data-cap-action="enable" data-cap-id="${escapeHTML(id)}" ${c.enabled ? "disabled" : ""}>Enable</button></div></article>`; }).join("") || `<div class="empty-state"><h2>No registered capabilities</h2><p>Bifrost does not discover or install capabilities automatically.</p></div>`; document.querySelectorAll("[data-cap-action]").forEach((b) => b.addEventListener("click", () => capabilityAction(b.dataset.capAction, b.dataset.capId))); } catch (error) { renderCapabilitiesUnavailable(error); } }
  async function loadConfiguration() { try { state.config = await json("/v1/gui/configuration"); renderConfiguration(); } catch (error) { showNotice(error.message, true); } }
  function renderConfiguration() { const fields = state.config?.fields || []; $("config-form").innerHTML = fields.map((f) => { const id = `config-${f.key.replaceAll(".", "-")}`; const value = f.value ?? ""; const input = typeof value === "boolean" ? `<input id="${id}" type="checkbox" ${value ? "checked" : ""} ${f.editable ? "" : "disabled"}>` : `<input id="${id}" type="${typeof value === "number" ? "number" : "text"}" value="${escapeHTML(value)}" ${f.min != null ? `min="${f.min}"` : ""} ${f.max != null ? `max="${f.max}"` : ""} ${f.editable ? "" : "disabled"}>`; return `<div class="config-field ${f.editable ? "" : "readonly"}"><label for="${id}">${escapeHTML(f.label)}${f.editable ? "" : " · read-only"}</label><small>${escapeHTML(f.description)} ${f.restart_required ? "Restart required after apply." : ""}</small>${input}</div>`; }).join(""); $("config-message").textContent = `${state.config.path} · secrets omitted`; $("config-message").classList.remove("error"); }
  function readConfigurationChanges() { const changes = {}; (state.config?.fields || []).filter((f) => f.editable).forEach((f) => { const input = $(`config-${f.key.replaceAll(".", "-")}`); if (!input) return; changes[f.key] = input.type === "checkbox" ? input.checked : (input.type === "number" ? Number(input.value) : input.value); }); return changes; }
  async function applyConfiguration() { const changes = readConfigurationChanges(); try { const validation = await post("/v1/gui/configuration/validate", { changes }); if (!validation.valid) throw new Error(validation.error || "Configuration validation failed"); const restart = validation.restart_required ? " Some changes require a daemon restart." : ""; if (!window.confirm(`Apply ${validation.changed?.length || 0} validated configuration changes?${restart}`)) return; const applied = await post("/v1/gui/configuration/apply", { changes }); showNotice(`Configuration applied. Backup: ${applied.backup || "created"}.${applied.restart_required ? " Restart is required for marked fields." : ""}`); await loadConfiguration(); await refreshState(); } catch (error) { $("config-message").textContent = error.message; $("config-message").classList.add("error"); } }
  async function controlPrometheus(paused) { const action = paused ? "pause" : "resume"; const consequence = paused ? "Pause Pantheon? New work will be refused until resumed; skipped work will not replay." : "Resume Pantheon? Work may be accepted again; the blocked interval will not replay."; if (!window.confirm(consequence)) return; try { await post(`/v1/gui/prometheus/${action}`); showNotice(paused ? "Prometheus paused Pantheon." : "Prometheus resumed Pantheon."); await refreshState(); } catch (error) { showNotice(error.message, true); } }
  async function rumorSourceAction(action, id) { try { await post(`/v1/gui/rumor/sources/${encodeURIComponent(id)}/${action}`); await loadRumor(); } catch (error) { showNotice(error.message, true); } }
  async function capabilityAction(action, id) { if (!window.confirm(`${action === "enable" ? "Enable" : "Disable"} this governed capability?`)) return; try { await post(`/v1/gui/bifrost/capabilities/${encodeURIComponent(id)}/${action}`); await loadCapabilities(); } catch (error) { showNotice(error.message, true); } }
  $("pause-button").addEventListener("click", () => controlPrometheus(true)); $("resume-button").addEventListener("click", () => controlPrometheus(false)); $("config-apply").addEventListener("click", applyConfiguration); $("config-cancel").addEventListener("click", renderConfiguration); $("refresh-notifications").addEventListener("click", loadNotifications); $("refresh-system").addEventListener("click", refreshState); $("execution-filter")?.addEventListener("change", (event) => loadExecutionWork(event.target.value)); $("execution-refresh")?.addEventListener("click", () => loadExecutionWork($("execution-filter")?.value || "all")); $("rumor-scan").addEventListener("click", async () => { try { await post("/v1/gui/rumor/scan"); await loadRumor(); showNotice("Rumor scan completed."); } catch (error) { showNotice(error.message, true); } }); $("mail-round")?.addEventListener("click", checkMail);
  $("zodiac-round")?.addEventListener("click", evaluateZodiac); $("zodiac-refresh")?.addEventListener("click", loadZodiac);
  $("aphrodite-refresh")?.addEventListener("click", loadAphrodite);
  $("echo-refresh")?.addEventListener("click", loadEcho);
  $("echo-run-select")?.addEventListener("change", (event) => loadEcho(event.target.value));
  $("fame-refresh")?.addEventListener("click", loadFame);
  function updateClock() { const now = new Date(); $("clock").textContent = now.toLocaleTimeString(); $("date").textContent = now.toLocaleDateString(undefined, { weekday: "long", year: "numeric", month: "long", day: "numeric" }); $("runtime-time").textContent = now.toLocaleTimeString(); }
  async function boot() { try { await post("/v1/gui/session"); await refreshState(); updateClock(); setInterval(updateClock, 1000); setInterval(refreshState, 3000); } catch (error) { showNotice("Panopticon could not establish a local session. Check that Pantheon is running and retry.", true); console.warn("[Panopticon] Session startup failed", error); } }
  boot();
})();
