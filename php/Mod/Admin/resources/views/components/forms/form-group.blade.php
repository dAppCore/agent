<div {{ $attributes->merge(['class' => 'space-y-2']) }}>
    @if($label)
        <div class="text-sm font-medium text-zinc-900">{{ $label }}@if($required) * @endif</div>
    @endif

    {{ $slot }}

    @if($help)
        <div class="text-xs text-zinc-500">{{ $help }}</div>
    @endif

    @if($error)
        <div class="text-sm text-red-600">{{ $error }}</div>
    @endif
</div>
