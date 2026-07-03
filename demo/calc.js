// demo JS file for testing the Jest test generator

// 纯函数
function add(a, b) {
  return a + b;
}

function divide(a, b) {
  if (b === 0) {
    throw new Error("division by zero");
  }
  return a / b;
}

// async 函数
async function fetchData(url) {
  const response = await fetch(url);
  return response.json();
}

// 箭头函数
const greet = (name) => {
  return `Hello, ${name}!`;
};

const multiply = (a, b) => a * b;

// 变参 + 默认值
function formatText(text, prefix = "", ...args) {
  return prefix + text + args.join("");
}

// 类
class Calculator {
  constructor() {
    this.history = [];
  }

  add(a, b) {
    const result = a + b;
    this.history.push(result);
    return result;
  }

  async divide(a, b) {
    if (b === 0) {
      throw new Error("division by zero");
    }
    return a / b;
  }

  clear() {
    this.history = [];
  }
}

// 导出
module.exports = { add, divide, fetchData, greet, multiply, formatText, Calculator };
