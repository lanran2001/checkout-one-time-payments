const stripe = Stripe('pk_test_51PGYMiRoUfb6BI4pnpQl3XO4VO43wlSMI0qb75vkFNg13pqhMjVdsUbL78hk28jVMUP3UdMBcIQULKZJ9NU3723w00t5zBcCvl');

async function setupPayment() {
    // 调用Go后端创建SetupIntent
    const response = await fetch('/create-setup-intent', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            CustomerId: 'cus_RazwBE0EYQAP7u',
        }),
    });
    const data = await response.json();

    const elements = stripe.elements(
        {clientSecret: data.clientSecret}
    );
    const paymentElement = elements.create('payment');
    paymentElement.mount('#payment-element');

    // 处理表单提交
    const form = document.getElementById('payment-form');
    form.addEventListener('submit', async (event) => {
        event.preventDefault();
        console.log('submit');
        const { setupIntent, error } = await stripe.confirmSetup({
            elements,
            redirect: 'if_required',
            // confirmParams: {
            //     return_url: 'https://example.com/setup-complete',
            // }
        });
        console.log(setupIntent);
        if (error) {
            console.error('Setup failed:', error);
            return
        } else {
            console.log('Setup succeeded:', setupIntent);
        }

        const response = await fetch('/save-payment-method', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                paymentMethodId: setupIntent.payment_method,
                setupIntentId: setupIntent.id,
                CustomerId: data.customerId,
            }),
        });
    
        console.log('Payment method saved', response);
    });
}

setupPayment();