const isEmailValid = (email: string): boolean => {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
};

const isPasswordValid = (password: string): boolean => {
    return password.length >= 3 && password.length <= 64;
};

export { isEmailValid, isPasswordValid };
