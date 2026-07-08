module.exports = {
  root: true,
  env: { browser: true, es2022: true, node: true },
  parser: 'vue-eslint-parser',
  parserOptions: { parser: '@typescript-eslint/parser', ecmaVersion: 'latest', sourceType: 'module' },
  extends: [
    'eslint:recommended',
    'plugin:vue/vue3-recommended',
    'plugin:@typescript-eslint/recommended',
    'prettier',
  ],
  rules: {
    'vue/multi-word-component-names': 'off',
    'vue/valid-define-props': 'off',
    'vue/valid-define-emits': 'off',
    '@typescript-eslint/no-explicit-any': 'warn',
    'no-constant-condition': ['error', { checkLoops: false }],
    'no-undef': 'off',
  },
}
