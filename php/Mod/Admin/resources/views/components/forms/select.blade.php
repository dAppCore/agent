<label class="block space-y-2">
    @if($label)
        <span class="text-sm font-medium text-zinc-900">{{ $label }}@if($required) * @endif</span>
    @endif

    <select
        id="{{ $id }}"
        @disabled($disabled)
        @required($required)
        @if($multiple) multiple @endif
        {{ $attributes->merge([
            'class' => 'w-full rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm text-zinc-900 shadow-sm focus:border-violet-500 focus:outline-none focus:ring-2 focus:ring-violet-200 disabled:bg-zinc-100 disabled:text-zinc-500',
        ]) }}
    >
        @foreach($options as $value => $optionLabel)
            <option value="{{ $value }}">{{ $optionLabel }}</option>
        @endforeach
    </select>
</label>
