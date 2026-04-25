<button
    type="{{ $type }}"
    @disabled($disabled)
    {{ $attributes->merge([
        'class' => 'inline-flex items-center justify-center gap-2 rounded-lg font-medium transition focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-60 '.$variantClasses.' '.$sizeClasses,
    ]) }}
>
    @if($loading)
        <span wire:loading.inline class="h-4 w-4 animate-spin rounded-full border-2 border-current border-r-transparent"></span>
    @endif

    @if($icon)
        <span aria-hidden="true">{{ $icon }}</span>
    @endif

    <span>{{ $slot }}</span>
</button>
