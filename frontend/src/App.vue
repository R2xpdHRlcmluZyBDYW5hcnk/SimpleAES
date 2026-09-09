<template>
  <div class="container">
    <header>
      <h1>SimpleAES</h1>
      <p class="caption">AES-256-GCM · PBKDF2-HMAC-SHA256</p>
    </header>

    <div class="radios">
      <label>
        <input type="radio" v-model="mode" value="encrypt" /> Encrypt
      </label>
      <label>
        <input type="radio" v-model="mode" value="decrypt" /> Decrypt
      </label>
    </div>

    <div class="iter-row">
      <span>PBKDF2 iterations:</span>
      <input
        v-model.number="iterations"
        type="number"
        :min="MIN_ITERS"
        :max="MAX_ITERS"
        step="1000"
      />
      <span class="caption">allowed range: {{ MIN_ITERS }} - {{ MAX_ITERS }}</span>
    </div>

    <textarea
      ref="contentRef"
      v-model="content"
      :placeholder="placeholderText"
      spellcheck="false"
      autocomplete="off"
      autocorrect="off"
      autocapitalize="off"
    ></textarea>

    <input
      ref="passwordRef"
      class="field"
      v-model="password"
      type="password"
      placeholder="Password (press Enter to submit)"
      autocomplete="off"
      @keydown.enter="perform"
    />

    <div
      class="status"
      :class="{ err: isError, ok: statusMsg && !isError }"
    >
      {{ statusMsg }}
    </div>

    <div class="buttons">
      <button class="action-btn" :disabled="isLoading" @click="perform">
        {{ isDecrypt ? "Decrypt" : "Encrypt" }}
      </button>
      <button :disabled="isLoading" @click="copyResult">
        Copy Result
      </button>
      <button :disabled="isLoading" @click="clearAll">
        Clear
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, nextTick } from "vue";
import { Encrypt, Decrypt } from "../bindings/SimpleAES/app";

const MIN_ITERS = 1000;
const MAX_ITERS = 10000000;
const DEFAULT_ITERS = 600000;

const mode = ref("encrypt");
const iterations = ref(DEFAULT_ITERS);
const content = ref("");
const password = ref("");
const statusMsg = ref("");
const isError = ref(false);
const isLoading = ref(false);

const contentRef = ref(null);
const passwordRef = ref(null);

const isDecrypt = computed(() => mode.value === "decrypt");

const placeholderText = computed(() =>
  isDecrypt.value
    ? "Paste Base64 ciphertext here (Decrypt mode)"
    : "Enter plaintext here (Encrypt mode)"
);

function setStatus(msg, err = false) {
  statusMsg.value = msg;
  isError.value = !!err;
}

watch(mode, () => {
  setStatus("");
});

async function perform() {
  if (!content.value.trim()) {
    return setStatus("Content is empty", true);
  }
  if (!password.value) {
    return setStatus("Password is empty", true);
  }

  const iters = Number(iterations.value);
  if (!Number.isInteger(iters) || iters < MIN_ITERS || iters > MAX_ITERS) {
    return setStatus(
      `Iterations must be an integer between ${MIN_ITERS} and ${MAX_ITERS}`,
      true
    );
  }

  isLoading.value = true;
  try {
    const result = isDecrypt.value
      ? await Decrypt(content.value, password.value)
      : await Encrypt(content.value, password.value, iters);

    content.value = result;
    await nextTick();
    if (contentRef.value) {
      contentRef.value.scrollTop = 0;
    }

    setStatus(
      isDecrypt.value
        ? "Decrypted successfully"
        : `Encrypted successfully (PBKDF2 ${iters} iterations)`,
      false
    );
    password.value = "";
  } catch (e) {
    setStatus(`${isDecrypt.value ? "Decryption" : "Encryption"} failed: ${e}`, true);
  } finally {
    isLoading.value = false;
  }
}

async function copyResult() {
  if (!content.value) return;
  try {
    await navigator.clipboard.writeText(content.value);
    setStatus("Result copied to clipboard", false);
  } catch (e) {
    setStatus(`Copy failed: ${e}`, true);
  }
}

function clearAll() {
  content.value = "";
  password.value = "";
  iterations.value = DEFAULT_ITERS;
  setStatus("");
  nextTick(() => {
    contentRef.value?.focus();
  });
}

onMounted(() => {
  contentRef.value?.focus();
});
</script>
