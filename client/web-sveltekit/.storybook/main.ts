import type { StorybookConfig } from '@storybook/sveltekit'

const config: StorybookConfig = {
    stories: ['../src/**/*.mdx', '../src/**/*.stories.@(js|jsx|ts|tsx)'],
    addons: [
        '@storybook/addon-links',
        '@storybook/addon-essentials',
        '@storybook/addon-interactions',
        'storybook-dark-mode',
    ],
    framework: {
        name: '@storybook/sveltekit',
        options: {},
    },
    docs: {
        autodocs: 'tag',
    },
}
export default config
