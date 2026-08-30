"use strict";

const $ = (id) => document.getElementById(id);

const contentEl = $("content");
const passwordEl = $("password");
const iterEl = $("iterations");
const statusEl = $("status");
const actionBtn = $("action");
const copyBtn = $("copy");
const clearBtn = $("clear");

const MIN_ITERS = 1000;
const MAX_ITERS = 10000000;
const DEFAULT_ITERS = 600000;

function mode() {
  return document.querySelector('input[name="mode"]:checked').value;
}

function setStatus(msg, isErr) {
  statusEl.textContent = msg;
  statusEl.classList.toggle("err", !!isErr);
  statusEl.classList.toggle("ok", !!msg && !isErr);
}

function refreshModeUI() {
  const decrypt = mode() === "decrypt";
  actionBtn.textContent = decrypt ? "Decrypt" : "Encrypt";
  contentEl.placeholder = decrypt
    ? "Paste Base64 ciphertext here (Decrypt mode)"
    : "Enter plaintext here (Encrypt mode)";
  setStatus("");
}

async function perform() {
  const content = contentEl.value;
  const password = passwordEl.value;
  if (!content.trim()) return setStatus("Content is empty", true);
  if (!password) return setStatus("Password is empty", true);

  const iterations = Number(iterEl.value);
  if (!Number.isInteger(iterations) || iterations < MIN_ITERS || iterations > MAX_ITERS) {
    return setStatus(
      `Iterations must be an integer between ${MIN_ITERS} and ${MAX_ITERS}`,
      true
    );
  }

  const decrypt = mode() === "decrypt";
  actionBtn.disabled = true;
  try {
    const result = decrypt
      ? await window.go.main.App.Decrypt(content, password)
      : await window.go.main.App.Encrypt(content, password, iterations);
    contentEl.value = result;
    contentEl.scrollTop = 0; // 新结果从顶部开始显示
    setStatus(
      decrypt
        ? "Decrypted successfully"
        : `Encrypted successfully (PBKDF2 ${iterations} iterations)`,
      false
    );
    passwordEl.value = "";
  } catch (e) {
    setStatus(`${decrypt ? "Decryption" : "Encryption"} failed: ${e}`, true);
  } finally {
    actionBtn.disabled = false;
  }
}

async function copyResult() {
  const text = contentEl.value;
  if (!text) return;
  try {
    await navigator.clipboard.writeText(text);
    setStatus("Result copied to clipboard", false);
  } catch (e) {
    setStatus(`Copy failed: ${e}`, true);
  }
}

function clearAll() {
  contentEl.value = "";
  passwordEl.value = "";
  iterEl.value = DEFAULT_ITERS;
  setStatus("");
  contentEl.focus();
}

document
  .querySelectorAll('input[name="mode"]')
  .forEach((el) => el.addEventListener("change", refreshModeUI));
actionBtn.addEventListener("click", perform);
copyBtn.addEventListener("click", copyResult);
clearBtn.addEventListener("click", clearAll);
passwordEl.addEventListener("keydown", (e) => {
  if (e.key === "Enter") perform();
});

refreshModeUI();
contentEl.focus();
