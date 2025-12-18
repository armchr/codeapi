/**
 * Operations module - barrel export file.
 */

// Re-export everything from types
export * from './types.js';

// Re-export from basic
export {
    add,
    subtract,
    multiply,
    divide,
    power,
    modulo,
    sum,
    product,
    getOperationBySymbol,
    createOperation,
    preciseAdd,
    preciseMultiply,
    preciseDivide,
    negate,
    absolute,
    square,
    sqrt,
    compose,
    pipe,
    map,
    filter,
    reduce,
    batchOperation,
} from './basic.js';

// Re-export default as named
export { default as basicOperations } from './basic.js';

// Re-export classes
export {
    BaseCalculator,
    AdvancedCalculator,
    ScientificCalculator,
    LoggingCalculator,
    ValidatingCalculator,
    LoggingValidatingCalculator,
    CalculatorSingleton,
} from './advanced.js';
