export const generateId = (type?: string): string => {
    return type + '_' + Math.random().toString(36).substr(2, 9);
};
