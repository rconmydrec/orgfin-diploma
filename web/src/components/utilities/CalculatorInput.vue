<template>
  <div
    ref="containerRef"
    class="calculator-input-wrapper">
    <div class="input-container">
      <input
        ref="inputRef"
        :id="inputId"
        v-model="displayValue"
        type="text"
        inputmode="decimal"
        class="form-control"
        :placeholder="placeholder"
        :disabled="disabled"
        :required="required"
        @input="handleInputChange"
        @keydown="handleKeyDown"
        @blur="handleInputBlur" />

      <!-- Calculator toggle button -->
      <button
        type="button"
        class="calculator-toggle-btn"
        :class="{ active: showCalculator }"
        :disabled="disabled"
        @click="toggleCalculator"
        @mousedown.prevent
        :aria-label="$t('calculator.toggle')">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          class="calculator-icon"
          viewBox="0 0 20 20"
          fill="currentColor">
          <path
            fill-rule="evenodd"
            d="M6 2a2 2 0 00-2 2v12a2 2 0 002 2h8a2 2 0 002-2V4a2 2 0 00-2-2H6zm1 2a1 1 0 000 2h6a1 1 0 100-2H7zm6 7a1 1 0 011 1v3a1 1 0 11-2 0v-3a1 1 0 011-1zm-3 3a1 1 0 100 2h.01a1 1 0 100-2H10zm-4 1a1 1 0 011-1h.01a1 1 0 110 2H7a1 1 0 01-1-1zm1-4a1 1 0 100 2h.01a1 1 0 100-2H7zm2 1a1 1 0 011-1h.01a1 1 0 110 2H10a1 1 0 01-1-1zm4-4a1 1 0 100 2h.01a1 1 0 100-2H13zM9 9a1 1 0 011-1h.01a1 1 0 110 2H10a1 1 0 01-1-1zM7 8a1 1 0 000 2h.01a1 1 0 000-2H7z"
            clip-rule="evenodd" />
        </svg>
      </button>
    </div>

    <!-- Calculator panel -->
    <transition name="calculator">
      <div
        v-if="showCalculator"
        class="calculator-panel"
        :style="calculatorStyle"
        @click.stop>
        <!-- Header with close button -->
        <div class="calculator-header">
          <span class="calculator-title">{{ $t('calculator.toggle') }}</span>
          <button
            type="button"
            class="calculator-close-btn"
            @click="closeCalculator"
            @mousedown.prevent
            :aria-label="$t('buttons.close')">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              class="close-icon"
              viewBox="0 0 20 20"
              fill="currentColor">
              <path
                fill-rule="evenodd"
                d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                clip-rule="evenodd" />
            </svg>
          </button>
        </div>

        <!-- Calculator buttons grid -->
        <div class="calculator-buttons">
          <template
            v-for="(row, rowIndex) in buttonLayout"
            :key="rowIndex">
            <button
              v-for="btn in row"
              :key="btn"
              type="button"
              class="calc-btn"
              :class="getButtonClass(btn)"
              @click="handleCalculatorButton(btn)"
              @mousedown.prevent>
              {{ btn }}
            </button>
          </template>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue';

const props = defineProps({
  modelValue: {
    type: [Number, String],
    default: '',
  },
  placeholder: {
    type: String,
    default: '0.00',
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  required: {
    type: Boolean,
    default: false,
  },
  inputId: {
    type: String,
    default: '',
  },
  min: {
    type: Number,
    default: null,
  },
  max: {
    type: Number,
    default: null,
  },
});

const emit = defineEmits(['update:modelValue']);

const inputRef = ref(null);
const containerRef = ref(null);
const showCalculator = ref(false);
const displayValue = ref('');
const calculatorPosition = ref({ top: 0, left: 0 });

// Calculator button layout (matching React version)
const buttonLayout = [
  ['7', '8', '9', '/'],
  ['4', '5', '6', '*'],
  ['1', '2', '3', '-'],
  ['0', '.', '=', '+'],
  ['C', '←', '(', ')'],
];

// Initialize display value from modelValue
watch(
  () => props.modelValue,
  (newValue) => {
    const stringValue = newValue ? String(newValue) : '';
    if (displayValue.value !== stringValue) {
      displayValue.value = stringValue;
    }
  },
  { immediate: true },
);

// Computed style for calculator positioning
const calculatorStyle = computed(() => {
  const pos = calculatorPosition.value;
  const style = {
    position: 'absolute',
    left: `${pos.left}px`,
    zIndex: 1050,
  };

  if (pos.top !== undefined) {
    style.top = `${pos.top}px`;
  } else if (pos.bottom !== undefined) {
    style.bottom = `${pos.bottom}px`;
  }

  return style;
});

// Get button CSS class based on button type
const getButtonClass = (btn) => {
  const operators = ['+', '-', '*', '/', '='];
  const special = ['C', '←'];

  if (operators.includes(btn)) {
    return 'calc-btn-operator';
  } else if (special.includes(btn)) {
    return 'calc-btn-special';
  }
  return 'calc-btn-number';
};

// Check if value contains math expression
const isMathExpression = (value) => {
  return /[+\-*/()]/.test(value);
};

// Safely evaluate mathematical expression
const evaluateExpression = (expr) => {
  try {
    // Remove all whitespace
    const cleanExpr = expr.replace(/\s/g, '');

    // Security: Only allow numbers, operators, dots, and parentheses
    if (!/^[\d+\-*/.()]+$/.test(cleanExpr)) {
      return null;
    }

    if (!cleanExpr) {
      return null;
    }

    // Use Function constructor for safer evaluation than eval

    const result = new Function(`'use strict'; return (${cleanExpr})`)();

    // Check if result is a valid number
    if (typeof result === 'number' && !isNaN(result) && isFinite(result)) {
      // Round to 2 decimal places
      return Math.round(result * 100) / 100;
    }

    return null;
  } catch (error) {
    return null;
  }
};

// Format calculator result
const formatCalculatorResult = (num) => {
  return num.toString();
};

// Handle input change
const handleInputChange = (event) => {
  const value = event.target.value;
  // Replace commas with dots and allow operators
  const normalizedValue = value.replace(/,/g, '.');

  // Allow numbers, operators, dots, and parentheses
  if (/^[\d+\-*/.()]*$/.test(normalizedValue) || normalizedValue === '') {
    displayValue.value = normalizedValue;
    emit('update:modelValue', normalizedValue);
  }
};

// Handle calculator button click
const handleCalculatorButton = (buttonValue) => {
  if (buttonValue === 'C') {
    // Clear
    displayValue.value = '';
    emit('update:modelValue', '');
  } else if (buttonValue === '←') {
    // Backspace
    displayValue.value = displayValue.value.slice(0, -1);
    emit('update:modelValue', displayValue.value);
  } else if (buttonValue === '=') {
    // Evaluate expression
    if (isMathExpression(displayValue.value)) {
      const result = evaluateExpression(displayValue.value);
      if (result !== null) {
        const formattedResult = formatCalculatorResult(result);
        displayValue.value = formattedResult;
        emit('update:modelValue', formattedResult);
      }
    }
  } else {
    // Append button value to current input
    displayValue.value += buttonValue;
    emit('update:modelValue', displayValue.value);
  }

  // Keep focus on input after button click
  nextTick(() => {
    if (inputRef.value) {
      inputRef.value.focus();
    }
  });
};

// Handle keyboard events
const handleKeyDown = (event) => {
  // When Enter is pressed, try to evaluate the expression
  if (event.key === 'Enter' && isMathExpression(displayValue.value)) {
    event.preventDefault();
    const result = evaluateExpression(displayValue.value);
    if (result !== null) {
      const formattedResult = formatCalculatorResult(result);
      displayValue.value = formattedResult;
      emit('update:modelValue', formattedResult);
    }
  }
};

// Handle input blur
const handleInputBlur = () => {
  // When input loses focus, evaluate if it's a math expression
  if (isMathExpression(displayValue.value)) {
    const result = evaluateExpression(displayValue.value);
    if (result !== null) {
      const formattedResult = formatCalculatorResult(result);
      displayValue.value = formattedResult;
      emit('update:modelValue', formattedResult);
    }
  }
};

// Update calculator position
const updateCalculatorPosition = () => {
  if (!containerRef.value) return;

  const rect = containerRef.value.getBoundingClientRect();
  const spaceBelow = window.innerHeight - rect.bottom;
  const spaceAbove = rect.top;

  // Decide whether to show calculator above or below the input
  if (spaceBelow > 320 || spaceBelow > spaceAbove) {
    // Show below
    calculatorPosition.value = {
      top: rect.height + 4,
      left: 0,
    };
  } else {
    // Show above
    calculatorPosition.value = {
      bottom: rect.height + 4,
      left: 0,
    };
  }
};

// Toggle calculator
const toggleCalculator = (e) => {
  e.preventDefault();
  e.stopPropagation();

  const newState = !showCalculator.value;
  showCalculator.value = newState;

  if (newState) {
    updateCalculatorPosition();
    // Focus input when calculator is opened
    nextTick(() => {
      if (inputRef.value) {
        inputRef.value.focus();
      }
    });
  }
};

// Close calculator
const closeCalculator = () => {
  showCalculator.value = false;
};

// Close calculator when clicking outside
const handleClickOutside = (event) => {
  if (
    showCalculator.value &&
    containerRef.value &&
    !containerRef.value.contains(event.target)
  ) {
    closeCalculator();
  }
};

onMounted(() => {
  document.addEventListener('click', handleClickOutside);
});

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside);
});

// Expose input ref for parent component access
defineExpose({ inputRef });
</script>

<style scoped>
.calculator-input-wrapper {
  position: relative;
  width: 100%;
}

.input-container {
  position: relative;
  display: flex;
  align-items: center;
}

.input-container input {
  padding-right: 2.5rem;
}

.calculator-toggle-btn {
  position: absolute;
  right: 0.5rem;
  top: 50%;
  transform: translateY(-50%);
  padding: 0.25rem;
  border: none;
  background: transparent;
  color: #6c757d;
  cursor: pointer;
  transition: all 0.15s ease-in-out;
  border-radius: 0.25rem;
}

.calculator-toggle-btn:hover {
  color: #495057;
  background: #f8f9fa;
}

.calculator-toggle-btn.active {
  color: #0d6efd;
  background: #e7f1ff;
}

.calculator-toggle-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.calculator-icon {
  width: 1.25rem;
  height: 1.25rem;
}

.calculator-panel {
  background: white;
  border-radius: 0.5rem;
  box-shadow: 0 0.5rem 1.5rem rgba(0, 0, 0, 0.2);
  border: 1px solid #dee2e6;
  padding: 0.75rem;
  width: 16rem;
}

.calculator-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid #e9ecef;
}

.calculator-title {
  font-size: 0.875rem;
  font-weight: 500;
  color: #495057;
}

.calculator-close-btn {
  padding: 0.25rem;
  border: none;
  background: transparent;
  color: #6c757d;
  cursor: pointer;
  transition: color 0.15s ease-in-out;
  border-radius: 0.25rem;
}

.calculator-close-btn:hover {
  color: #495057;
}

.close-icon {
  width: 1.25rem;
  height: 1.25rem;
}

.calculator-buttons {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 0.5rem;
}

.calc-btn {
  padding: 0.5rem;
  border: 1px solid #dee2e6;
  border-radius: 0.25rem;
  background: white;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease-in-out;
  color: #495057;
}

.calc-btn:hover {
  background: #f8f9fa;
}

.calc-btn:active {
  transform: scale(0.95);
}

.calc-btn-number {
  background: #f8f9fa;
  color: #212529;
}

.calc-btn-operator {
  background: #0d6efd;
  color: white;
  border-color: #0d6efd;
}

.calc-btn-operator:hover {
  background: #0b5ed7;
  border-color: #0a58ca;
}

.calc-btn-special {
  background: #dc3545;
  color: white;
  border-color: #dc3545;
}

.calc-btn-special:hover {
  background: #bb2d3b;
  border-color: #b02a37;
}

/* Transition animations */
.calculator-enter-active,
.calculator-leave-active {
  transition: all 0.2s ease;
}

.calculator-enter-from {
  opacity: 0;
  transform: translateY(-0.5rem);
}

.calculator-leave-to {
  opacity: 0;
  transform: translateY(-0.5rem);
}
</style>
