import { describe, it } from '@std/testing/bdd';
import { CssDocument, getLightDarkValues } from '#css';
import { expect } from '@std/expect/expect';
import { createTestContext } from '#test-helpers';

describe('getLightDarkValues', () => {
  it('should return light and dark values for a given value', () => {
    const value = 'light-dark(red, maroon)';
    const [lightValue, darkValue] = getLightDarkValues(value);
    expect(lightValue).toBe('red');
    expect(darkValue).toBe('maroon');
  });

  it('should return an empty list for invalid value', () => {
    expect(getLightDarkValues('')).toEqual([]);
  });
});

const ctx = await createTestContext({
  testTokensSpecs: [
    {
      prefix: 'token',
      spec: '/tokens.json',
      tokens: {
        color: {
          red: {
            _: {
              $value: '#ff0000',
              $type: 'color',
            },
            hex: {
              $value: '#ff0000',
              $type: 'color',
            },
          },
        },
        space: {
          small: {
            $value: '4px',
            $type: 'size',
          },
        },
        font: {
          weight: {
            thin: {
              $value: 100,
              $type: 'fontWeight',
            },
          },
        },
      },
    },
  ],
});

describe('CssDocument', () => {
  it('should create a CssDocument instance', () => {
    const uri = 'file:///test.css';
    const languageId = 'css';
    const version = 1;
    const text = 'body { color: red; }';

    const doc = CssDocument.create(ctx, uri, text, version);

    expect(doc.uri).toEqual(uri);
    expect(doc.languageId).toEqual(languageId);
    expect(doc.version).toEqual(version);
    expect(doc.getText()).toEqual(text);
    expect(doc.getFullRange()).toEqual({
      start: { line: 0, character: 0 },
      end: { line: 0, character: 20 },
    });
  });
});
